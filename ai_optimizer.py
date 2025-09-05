#!/usr/bin/env python3
"""
ai_optimizer.py

Usage:
  python ai_optimizer.py --port=4321 --username=root --password=admin [--duration=300] [--reset] [--prompt]

What it does:
 - Downloads all rows from system_statistic.scans via POST SQL query.
 - Tokenizes filter and order expressions into token sequences.
 - Encodes token features (36 dims per token: 16 hashed + 2 numeric + 1 paren depth + 17 operator flags) and feeds them into a transformer-based InputEncoder.
 - Passes encoder output through BaseModel, TableHistogram, optionally OrderedLayer.
 - Trains to predict outputCount (log1p(outputCount) for stability).
 - Periodically prints training loss.
 - Serializes models (base_encoder, ordered_layer, output_head, per-table histograms) to database.
 - Interactive mode (--prompt): load models from MemCP and predict outputCount for user-entered (schema, table, filter, order, ordered, inputCount) repeatedly.
"""

import argparse
import base64
import io
import json
import math
import re
import sys
import time
import mmap
import struct
from typing import List

import requests
import torch
import torch.nn as nn
import torch.optim as optim
from torch.utils.data import Dataset, DataLoader

# ---------- Config ----------
DEFAULT_PORT = 4321
DEFAULT_USER = "root"
DEFAULT_PASSWORD = "admin"
SQL_ENDPOINT_TEMPLATE = "http://localhost:{port}/sql/system_statistic"
BATCH_SIZE = 32
EPOCHS = 6
PRINT_EVERY = 50
LR = 1e-5
MAX_TOKENS = 512
EMBED_DIM = 256
TRANSFORMER_LAYERS = 4
TRANSFORMER_HEADS = 8
FF_DIM = 512
BASE_MLP_HIDDEN = 512
TABLE_HIST_HIDDEN = 128
ORDERED_LAYER_HIDDEN = 128
OUTPUT_HIDDEN = 128
RATIO_LOSS_WEIGHT = 0.3
SEED = 42
DEBUG = False

torch.manual_seed(SEED)

# ---------- Tokenizer & Features ----------
TOKEN_RE = re.compile(r"\(|\)|\?|>=|<=|<>|!=|=|>|<|\w+|\||\"(?:[^\"\\]|\\.)*\"", re.ASCII)
# Operator/keyword flags recognized by the model
OP_TOKENS = [
    "(", ")", "and", "or", "not",
    ">", "<", ">=", "<=", "=", "!=", "<>",
    "equal?", "equal??", "?", "true", "false",
]
FEATURE_DIM = 16 + 2 + 1 + len(OP_TOKENS)  # hash16 + num2 + parenDepth + operator flags

def tokenize_filter_order(filter_s: str, order_s: str) -> List[str]:
    tokens: List[str] = []
    if filter_s:
        tokens.extend(TOKEN_RE.findall(filter_s))
    if order_s:
        tokens.extend(str(order_s).split("|"))
    return tokens

def simple_hash_16(token: str, dim: int = 16) -> torch.Tensor:
    v = torch.zeros(dim, dtype=torch.float32)
    for i, c in enumerate(token):
        v[i % dim] += float(ord(c) & 0xFF) / 128.0
    norm = max(1.0, len(token) / dim)
    return v / norm

def token_features(tokens: List[str]) -> torch.Tensor:
    """Create FEATURE_DIM per token: hash16 + [num, log1p(num)] + parenDepth + operator flags."""
    rows: List[torch.Tensor] = []
    depth = 0
    for t in tokens:
        # Compute features for t; update depth after '('
        if t == ")" and depth > 0:
            # depth reflects the nesting at this token position
            pass
        h = simple_hash_16(t)
        # numeric
        val = float(int(t)) if re.fullmatch(r"-?\d+", t) else 0.0
        num = torch.tensor([val, math.log1p(abs(val)) if val != 0.0 else 0.0], dtype=torch.float32)
        # paren depth (scaled)
        depth_feat = torch.tensor([float(depth)/16.0], dtype=torch.float32)
        # operator flags
        flags = torch.zeros(len(OP_TOKENS), dtype=torch.float32)
        lt = t.lower()
        try:
            idx = OP_TOKENS.index(lt)
            flags[idx] = 1.0
        except ValueError:
            # treat "+-*/" etc via hashing only
            pass
        feat = torch.cat([h, num, depth_feat, flags], dim=0)
        rows.append(feat)
        if t == "(":
            depth += 1
        elif t == ")" and depth > 0:
            depth -= 1
    if not rows:
        rows = [torch.zeros(FEATURE_DIM, dtype=torch.float32)]
    return torch.stack(rows, dim=0)

class TokenProjector(nn.Module):
    def __init__(self, embed_dim: int = EMBED_DIM):
        super().__init__()
        self.linear = nn.Linear(FEATURE_DIM, embed_dim)
    def forward(self, feats: torch.Tensor) -> torch.Tensor:
        return self.linear(feats)


# ---------- Dataset ----------
class ScanRow:
    def __init__(self, schema, table, ordered, filter_s, order_s, inputCount, outputCount):
        self.schema = schema
        self.table = table
        self.ordered = bool(ordered)
        self.filter_s = filter_s or ""
        self.order_s = order_s or ""
        self.inputCount = float(inputCount) if inputCount is not None else 0
        self.outputCount = float(outputCount) if outputCount is not None else 0

class ScansDataset(Dataset):
    def __init__(self, rows: List[ScanRow], max_tokens: int = MAX_TOKENS):
        self.rows = rows
        self.max_tokens = max_tokens
        self.table_keys = sorted({(r.schema, r.table) for r in rows})
        self.table_to_idx = {k: i for i, k in enumerate(self.table_keys)}

    def __len__(self):
        return len(self.rows)

    def __getitem__(self, idx):
        r = self.rows[idx]
        # Derive ordered from order string (non-empty => ordered)
        derived_order = r.order_s or ""
        tokens = ["<CLS>"] + tokenize_filter_order(r.filter_s, derived_order) + ["<SEP>"]
        table_idx = self.table_to_idx[(r.schema, r.table)]
        return {
            "tokens": tokens,
            "table_idx": torch.tensor(table_idx, dtype=torch.long),
            "ordered": torch.tensor(1.0 if bool(derived_order) else 0.0, dtype=torch.float32),
            "inputCount": torch.tensor(r.inputCount, dtype=torch.float32),
            "outputCount": torch.tensor(r.outputCount, dtype=torch.float32),
            "schema": r.schema,
            "table": r.table,
        }

def collate_fn(batch):
    return batch  # we handle token embedding in training loop

# ---------- Model ----------
class PositionalEncoding(nn.Module):
    def __init__(self, d_model: int, max_len: int = MAX_TOKENS):
        super().__init__()
        pe = torch.zeros(max_len, d_model)
        position = torch.arange(0, max_len, dtype=torch.float32).unsqueeze(1)
        div_term = torch.exp(torch.arange(0, d_model, 2).float() * -(math.log(10000.0) / d_model))
        pe[:, 0::2] = torch.sin(position * div_term)
        pe[:, 1::2] = torch.cos(position * div_term)
        self.register_buffer("pe", pe.unsqueeze(0))
    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return x + self.pe[:, :x.size(1), :]

class TransformerBlock(nn.Module):
    def __init__(self, d_model=EMBED_DIM, nhead=TRANSFORMER_HEADS, dim_ff=FF_DIM, dropout=0.0):
        super().__init__()
        self.mha = nn.MultiheadAttention(d_model, nhead, batch_first=True)
        self.ff = nn.Sequential(
            nn.Linear(d_model, dim_ff),
            nn.ReLU(),
            nn.Linear(dim_ff, d_model),
        )
        self.norm1 = nn.LayerNorm(d_model)
        self.norm2 = nn.LayerNorm(d_model)
        self.dropout = nn.Dropout(dropout)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        attn_out, _ = self.mha(x, x, x)
        x = self.norm1(x + self.dropout(attn_out))
        ff_out = self.ff(x)
        x = self.norm2(x + self.dropout(ff_out))
        return x

class InputEncoder(nn.Module):
    def __init__(self, embed_dim=EMBED_DIM, nlayers=TRANSFORMER_LAYERS):
        super().__init__()
        self.pos_enc = PositionalEncoding(embed_dim, max_len=MAX_TOKENS)
        self.blocks = nn.ModuleList([TransformerBlock(embed_dim) for _ in range(nlayers)])
        self.pool = nn.Linear(embed_dim, embed_dim)

    def forward(self, token_vectors: torch.Tensor):
        x = self.pos_enc(token_vectors)
        for blk in self.blocks:
            x = blk(x)
        pooled = x.mean(dim=1)
        return self.pool(pooled)

class BaseModel(nn.Module):
    def __init__(self, embed_dim=EMBED_DIM):
        super().__init__()
        self.mlp = nn.Sequential(
            nn.Linear(embed_dim + 1, BASE_MLP_HIDDEN),
            nn.ReLU(),
            nn.Linear(BASE_MLP_HIDDEN, embed_dim),
            nn.ReLU(),
        )
    def forward(self, encoded_vec, inputCount_scalar):
        x = torch.cat([encoded_vec, inputCount_scalar.unsqueeze(-1).log1p() / 10.0], dim=-1)
        return self.mlp(x)

class BaseEncoderScript(nn.Module):
    """Scriptable base encoder pipeline used at inference time."""
    def __init__(self, embed_dim=EMBED_DIM):
        super().__init__()
        self.token_proj = nn.Linear(FEATURE_DIM, embed_dim)
        self.schema_proj = nn.Linear(FEATURE_DIM, embed_dim)
        self.table_proj = nn.Linear(FEATURE_DIM, embed_dim)
        self.encoder = InputEncoder(embed_dim=embed_dim)
        self.base_model = BaseModel(embed_dim=embed_dim)

    def forward(self, token_feats: torch.Tensor, schema_feats: torch.Tensor, table_feats: torch.Tensor, inputCount_scalar: torch.Tensor) -> torch.Tensor:
        # token_feats: (batch, seq, FEATURE_DIM)
        # schema_feats/table_feats: (batch, FEATURE_DIM)
        tok = self.token_proj(token_feats)
        enc = self.encoder(tok)
        sch = self.schema_proj(schema_feats)
        tab = self.table_proj(table_feats)
        combined = enc + sch + tab
        return self.base_model(combined, inputCount_scalar)

class TableHistogram(nn.Module):
    def __init__(self, embed_dim=EMBED_DIM):
        super().__init__()
        self.mlp = nn.Sequential(
            nn.Linear(embed_dim, TABLE_HIST_HIDDEN),
            nn.ReLU(),
            nn.Linear(TABLE_HIST_HIDDEN, embed_dim),
            nn.ReLU(),
        )
    def forward(self, x):
        return self.mlp(x)

class OrderedLayer(nn.Module):
    def __init__(self, embed_dim=EMBED_DIM):
        super().__init__()
        self.mlp = nn.Sequential(
            nn.Linear(embed_dim, ORDERED_LAYER_HIDDEN),
            nn.ReLU(),
            nn.Linear(ORDERED_LAYER_HIDDEN, embed_dim),
            nn.ReLU(),
        )
    def forward(self, x):
        return self.mlp(x)

class OutputHead(nn.Module):
    def __init__(self, embed_dim=EMBED_DIM):
        super().__init__()
        self.net = nn.Sequential(
            nn.Linear(embed_dim, OUTPUT_HIDDEN),
            nn.ReLU(),
            nn.Linear(OUTPUT_HIDDEN, 1),
        )
    def forward(self, x):
        return self.net(x).squeeze(-1)

class RatioHead(nn.Module):
    def __init__(self, embed_dim=EMBED_DIM):
        super().__init__()
        self.net = nn.Sequential(
            nn.Linear(embed_dim, OUTPUT_HIDDEN),
            nn.ReLU(),
            nn.Linear(OUTPUT_HIDDEN, 1),
        )
    def forward(self, x):
        return self.net(x).squeeze(-1)

# ---------- Full Training Model ----------
class FullTrainingModel(nn.Module):
    def __init__(self, n_tables, embed_dim=EMBED_DIM):
        super().__init__()
        self.base_encoder = BaseEncoderScript(embed_dim=embed_dim)
        self.table_histograms = nn.ModuleList([TableHistogram(embed_dim=embed_dim) for _ in range(n_tables)])
        self.ordered_layer = OrderedLayer(embed_dim=embed_dim)
        self.output_head = OutputHead(embed_dim=embed_dim)
        self.ratio_head = RatioHead(embed_dim=embed_dim)

    def forward(self, token_feats: torch.Tensor, schema_feats: torch.Tensor, table_feats: torch.Tensor,
                table_idx: torch.Tensor, ordered_flag: torch.Tensor, inputCount_scalar: torch.Tensor):
        base = self.base_encoder(token_feats, schema_feats, table_feats, inputCount_scalar)
        out = torch.zeros_like(base)
        unique_tables = torch.unique(table_idx)
        for t in unique_tables:
            mask = (table_idx == t)
            if mask.sum() > 0:
                out[mask] = self.table_histograms[int(t)](base[mask])
        if ordered_flag.sum() > 0:
            ord_mask = ordered_flag.bool()
            if ord_mask.any():
                out[ord_mask] = self.ordered_layer(out[ord_mask])
        # Return both predicted log1p(output) and ratio (log1p(output) - log1p(input))
        logp = self.output_head(out)
        ratio = self.ratio_head(out)
        return logp, ratio

class OrderedLayerScript(nn.Module):
    def __init__(self, embed_dim=EMBED_DIM):
        super().__init__()
        self.layer = OrderedLayer(embed_dim=embed_dim)
    def forward(self, x: torch.Tensor):
        return self.layer(x)

class OutputHeadScript(nn.Module):
    def __init__(self, embed_dim=EMBED_DIM):
        super().__init__()
        self.head = OutputHead(embed_dim=embed_dim)
    def forward(self, x: torch.Tensor):
        return self.head(x)


# ---------- Networking ----------
def fetch_scans(port: int, username: str, password: str) -> List[ScanRow]:
    url = f"http://localhost:{port}/sql/system_statistic"
    sql = "SELECT schema, table, ordered, filter, `order`, inputCount, outputCount FROM scans"
    rows: List[ScanRow] = []
    try:
        resp = requests.post(url, data=sql, auth=(username, password), stream=True, timeout=30)
        resp.raise_for_status()
        for line in resp.iter_lines(decode_unicode=True):
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                print(f"Skipping non-JSON line: {line[:80]!r}")
                continue
            rows.append(ScanRow(
                schema=obj.get("schema"),
                table=obj.get("table"),
                ordered=obj.get("ordered", False),
                filter_s=obj.get("filter"),
                order_s=obj.get("order"),
                inputCount=obj.get("inputCount", 0),
                outputCount=obj.get("outputCount", 0),
            ))
    except Exception as e:
        print(f"Warning: failed to fetch scans: {e}")
    return rows

def _load_script_by_id(port: int, username: str, password: str, model_id: str):
    obj = _first_json_row(port, username, password, f"SELECT model FROM base_models WHERE id='{model_id.replace("'","''")}' LIMIT 1")
    if not obj or 'model' not in obj:
        raise RuntimeError(f"Model {model_id} not found")
    b = _decode_model_field(obj['model'])
    if not b:
        raise RuntimeError(f"Model {model_id} empty/undecodable")
    return torch.jit.load(io.BytesIO(b), map_location='cpu')

def _load_hist_for_table(port: int, username: str, password: str, schema: str, table: str):
    sql = (
        "SELECT model FROM table_histogram WHERE `schema`='" + schema.replace("'","''") +
        "' AND `table`='" + table.replace("'","''") + "' LIMIT 1"
    )
    obj = _first_json_row(port, username, password, sql)
    if not obj or 'model' not in obj:
        raise RuntimeError(f"Histogram for {schema}.{table} not found")
    b = _decode_model_field(obj['model'])
    if not b:
        raise RuntimeError(f"Histogram for {schema}.{table} empty/undecodable")
    return torch.jit.load(io.BytesIO(b), map_location='cpu')

def _predict_one(base_encoder, ordered_layer, output_head, hist_layer, schema: str, table: str, filter_s: str, order_s: str, input_count: float) -> float:
    # Build features
    tokens = tokenize_filter_order(filter_s or "", order_s or "")
    feats = token_features(tokens)
    if feats.size(0) < MAX_TOKENS:
        pad = torch.zeros((MAX_TOKENS - feats.size(0), FEATURE_DIM), dtype=torch.float32)
        feats = torch.cat([feats, pad], dim=0)
    else:
        feats = feats[:MAX_TOKENS]
    tok = feats.unsqueeze(0)  # (1, T, F)
    sch = token_features([schema]).squeeze(0).unsqueeze(0)  # (1, F)
    tab = token_features([table]).squeeze(0).unsqueeze(0)   # (1, F)
    inC = torch.tensor([float(input_count)], dtype=torch.float32)

    with torch.no_grad():
        base_vec = base_encoder(tok, sch, tab, inC)
        out = hist_layer(base_vec)
        if bool(order_s):
            out = ordered_layer(out)
        logp = output_head(out)
        val = float(torch.expm1(logp).item())
    return max(0.0, val)

def run_prompt_loop(port: int, username: str, password: str):
    print("Loading models from MemCP...")
    try:
        base_encoder = _load_script_by_id(port, username, password, 'base_encoder_v1')
        ordered_layer = _load_script_by_id(port, username, password, 'ordered_layer_v1')
        output_head  = _load_script_by_id(port, username, password, 'output_head_v1')
    except Exception as e:
        print(f"Error: {e}")
        return
    print("Ready. Press Enter on schema to exit.")
    while True:
        try:
            schema = input("schema: ").strip()
        except EOFError:
            break
        if not schema:
            break
        table = input("table: ").strip()
        filter_s = input("filter (e.g., true or (and (> COL 10) (equal? 1 ?))): ").strip()
        order_s  = input("order  (e.g., C_CUSTKEY|C_NAME) [empty for unordered]: ").strip()
        inc_s    = input("inputCount (int): ").strip()
        try:
            input_count = float(inc_s)
        except Exception:
            print("invalid inputCount; try again")
            continue
        try:
            hist_layer = _load_hist_for_table(port, username, password, schema, table)
        except Exception as e:
            print(f"Error loading histogram for {schema}.{table}: {e}")
            continue
        try:
            pred = _predict_one(base_encoder, ordered_layer, output_head, hist_layer, schema, table, filter_s, order_s, input_count)
            print(f"predicted outputCount: {pred:.0f}")
        except Exception as e:
            print(f"prediction error: {e}")

def run_ipc_loop(ipc_path: str, port: int, username: str, password: str):
    # Load core models once; if unavailable, fall back to passthrough (output = input)
    base_encoder = None
    ordered_layer = None
    output_head  = None
    try:
        base_encoder = _load_script_by_id(port, username, password, 'base_encoder_v1')
        ordered_layer = _load_script_by_id(port, username, password, 'ordered_layer_v1')
        output_head  = _load_script_by_id(port, username, password, 'output_head_v1')
    except Exception as e:
        print(f"AIEstimator(py): core models not available ({e}); running passthrough mode")
    # Cache per-table histogram modules
    hist_cache: dict[tuple[str,str], torch.jit.ScriptModule] = {}
    with open(ipc_path, 'r+b') as f:
        mm = mmap.mmap(f.fileno(), 0, access=mmap.ACCESS_WRITE)
        try:
            # Status helpers
            def write_status(code: int, message: str = ""):
                # code: 1=READY, 2=INFO, 3=ERROR
                try:
                    msgb = message.encode('utf-8') if message else b''
                    # Write payload to status region (header+reqMax)
                    base = 64 + 512*1024
                    if len(msgb) > 64*1024:
                        msgb = msgb[:64*1024]
                    if msgb:
                        mm[base:base+len(msgb)] = msgb
                    # Bump statusSeq and publish
                    prev = struct.unpack_from('<Q', mm, 48)[0]
                    struct.pack_into('<I', mm, 56, len(msgb))
                    struct.pack_into('<I', mm, 60, int(code))
                    struct.pack_into('<Q', mm, 48, int(prev+1))
                except Exception:
                    pass

            write_status(2, "python: up and listening")
            last_seq = 0
            # Try loading base models; report status
            while True:
                # Emit READY once base load work is done
                if last_seq == 0:
                    try:
                        if base_encoder is not None and output_head is not None:
                            write_status(2, "python: core models loaded")
                        else:
                            write_status(2, "python: core models missing; passthrough mode")
                    except Exception as e:
                        write_status(3, f"core models error: {e}")
                    write_status(1, "ready")
                    last_seq = -1  # mark ready announced
                # Read reqSeq
                reqSeq = struct.unpack_from('<Q', mm, 0)[0]
                if reqSeq != 0 and reqSeq != last_seq:
                    # Read header
                    inputCount = struct.unpack_from('<Q', mm, 16)[0]
                    schemaLen, tableLen, filterLen, orderLen = struct.unpack_from('<IIII', mm, 24)
                    off = 64
                    if schemaLen == 0 and tableLen == 0 and filterLen == 0 and orderLen == 0:
                        # Generic opcode mode: [op:1][len:4][payload]
                        op = mm[off]
                        off += 1
                        plen = struct.unpack_from('<I', mm, off)[0]
                        off += 4
                        payload = mm[off:off+plen]
                        # Handle opcodes
                        if op == 2:  # FetchModel by ID (base model)
                            model_id = payload.decode('utf-8', 'ignore')
                            try:
                                obj = _first_json_row(port, username, password,
                                    "SELECT model FROM base_models WHERE id='" + model_id.replace("'","''") + "' LIMIT 1")
                                data = b''
                                if obj and 'model' in obj:
                                    b = _decode_model_field(obj['model'])
                                    if b:
                                        data = b
                                # Respond: [len][bytes]
                                struct.pack_into('<I', mm, 64, len(data))
                                if data:
                                    mm[68:68+len(data)] = data
                            except Exception as e:
                                write_status(3, f"fetch model error: {e}")
                                struct.pack_into('<I', mm, 64, 0)
                            struct.pack_into('<Q', mm, 8, int(reqSeq))
                            last_seq = reqSeq
                        elif op == 3:  # SQL text via system_statistic endpoint
                            sql = payload.decode('utf-8', 'ignore')
                            outb = b''
                            try:
                                txt = exec_sql(port, username, password, sql)
                                outb = txt.encode('utf-8')
                            except Exception as e:
                                msg = f"sql error: {e}"
                                write_status(3, msg)
                                outb = msg.encode('utf-8')
                            struct.pack_into('<I', mm, 64, len(outb))
                            if outb:
                                mm[68:68+len(outb)] = outb
                            struct.pack_into('<Q', mm, 8, int(reqSeq))
                            last_seq = reqSeq
                        else:
                            # Unknown opcode: respond empty
                            struct.pack_into('<I', mm, 64, 0)
                            struct.pack_into('<Q', mm, 8, int(reqSeq))
                            last_seq = reqSeq
                    else:
                        # Legacy estimate mode: payload is schema|table|filter|order
                        schema = mm[off:off+schemaLen].decode('utf-8', 'ignore'); off += schemaLen
                        table  = mm[off:off+tableLen].decode('utf-8', 'ignore');  off += tableLen
                        filter_s = mm[off:off+filterLen].decode('utf-8', 'ignore'); off += filterLen
                        order_s  = mm[off:off+orderLen].decode('utf-8', 'ignore');  off += orderLen
                        # Load histogram
                        key = (schema, table)
                        hist = hist_cache.get(key)
                        if hist is None:
                            try:
                                hist = _load_hist_for_table(port, username, password, schema, table)
                                hist_cache[key] = hist
                                write_status(2, f"loaded histogram {schema}.{table}")
                            except Exception:
                                hist = None
                        # Predict or fallback
                        if (base_encoder is None) or (output_head is None) or (hist is None):
                            out = float(inputCount)
                        else:
                            try:
                                out = _predict_one(base_encoder, ordered_layer, output_head, hist, schema, table, filter_s, order_s, float(inputCount))
                            except Exception:
                                out = float(inputCount)
                        # Write outputCount directly as float64 then respSeq
                        struct.pack_into('<d', mm, 40, float(out))
                        struct.pack_into('<Q', mm, 8, int(reqSeq))
                        last_seq = reqSeq
                else:
                    time.sleep(0.0002)
        finally:
            mm.close()

def _first_json_row(port: int, username: str, password: str, sql: str):
    url = SQL_ENDPOINT_TEMPLATE.format(port=port)
    try:
        # Non-streaming to avoid chunking issues on large base64 fields
        resp = requests.post(url, data=sql, auth=(username, password), timeout=120)
        resp.raise_for_status()
        text = resp.text
        lines = [ln for ln in text.splitlines() if ln.strip()]
        if not lines:
            return None
        # Use first non-empty JSONL line
        try:
            return json.loads(lines[0])
        except json.JSONDecodeError:
            if DEBUG:
                print(f"DEBUG: failed to parse JSON from first line, trying last line; text prefix={text[:80]!r}")
            try:
                return json.loads(lines[-1])
            except Exception:
                return None
    except Exception as e:
        print(f"Warning: SQL fetch failed: {e} sql={sql[:80]!r}")
    return None

def _decode_model_field(field) -> bytes | None:
    if field is None:
        return None
    if isinstance(field, (bytes, bytearray)):
        return bytes(field)
    if isinstance(field, str):
        try:
            return base64.b64decode(field, validate=True)
        except Exception:
            # Retry without strict validation (line breaks, url-safe, etc.)
            try:
                return base64.b64decode(field)
            except Exception:
                # Not base64; treat as raw text bytes
                try:
                    return field.encode('utf-8')
                except Exception:
                    return None
    return None

def load_existing_models(port: int, username: str, password: str, model: nn.Module, table_keys: list[tuple[str,str]]):
    """Load weights from DB if present into the eager training model.
    - Base encoder: base_encoder_v1
    - Ordered layer: ordered_layer_v1 (strip 'layer.' prefix)
    - Output head: output_head_v1 (strip 'head.' prefix)
    - Ratio head: ratio_head_v1
    - Table histograms: system_statistic.table_histogram per (schema, table)
    """
    # Base encoder
    obj = _first_json_row(port, username, password,
        "SELECT model FROM base_models WHERE id='base_encoder_v1' LIMIT 1")
    if obj and 'model' in obj:
        b = _decode_model_field(obj['model'])
        if b:
            try:
                sm = torch.jit.load(io.BytesIO(b), map_location='cpu')
                model.base_encoder.load_state_dict(sm.state_dict())
                if DEBUG:
                    print(f"DEBUG: loaded base_encoder_v1 bytes={len(b)}")
                else:
                    print("Loaded base_encoder_v1 from DB")
            except Exception as e:
                print(f"Warning: failed to load base_encoder_v1: {e}")

    # Ordered layer
    obj = _first_json_row(port, username, password,
        "SELECT model FROM base_models WHERE id='ordered_layer_v1' LIMIT 1")
    if obj and 'model' in obj:
        b = _decode_model_field(obj['model'])
        if b:
            try:
                sm = torch.jit.load(io.BytesIO(b), map_location='cpu')
                sd = {}
                for k, v in sm.state_dict().items():
                    if k.startswith('layer.'):
                        sd[k[len('layer.'):]] = v
                model.ordered_layer.load_state_dict(sd)
                if DEBUG:
                    print(f"DEBUG: loaded ordered_layer_v1 bytes={len(b)}")
                else:
                    print("Loaded ordered_layer_v1 from DB")
            except Exception as e:
                print(f"Warning: failed to load ordered_layer_v1: {e}")

    # Output head
    obj = _first_json_row(port, username, password,
        "SELECT model FROM base_models WHERE id='output_head_v1' LIMIT 1")
    if obj and 'model' in obj:
        b = _decode_model_field(obj['model'])
        if b:
            try:
                sm = torch.jit.load(io.BytesIO(b), map_location='cpu')
                sd = {}
                for k, v in sm.state_dict().items():
                    if k.startswith('head.'):
                        sd[k[len('head.'):]] = v
                model.output_head.load_state_dict(sd)
                if DEBUG:
                    print(f"DEBUG: loaded output_head_v1 bytes={len(b)}")
                else:
                    print("Loaded output_head_v1 from DB")
            except Exception as e:
                print(f"Warning: failed to load output_head_v1: {e}")

    # Ratio head
    obj = _first_json_row(port, username, password,
        "SELECT model FROM base_models WHERE id='ratio_head_v1' LIMIT 1")
    if obj and 'model' in obj:
        b = _decode_model_field(obj['model'])
        if b:
            try:
                sm = torch.jit.load(io.BytesIO(b), map_location='cpu')
                model.ratio_head.load_state_dict(sm.state_dict())
                if DEBUG:
                    print(f"DEBUG: loaded ratio_head_v1 bytes={len(b)}")
                else:
                    print("Loaded ratio_head_v1 from DB")
            except Exception as e:
                print(f"Warning: failed to load ratio_head_v1: {e}")

    # Table histograms per (schema, table)
    for idx, (schema, table) in enumerate(table_keys):
        sql = (
            "SELECT model FROM table_histogram WHERE `schema`='" + schema.replace("'","''") +
            "' AND `table`='" + table.replace("'","''") + "' LIMIT 1"
        )
        obj = _first_json_row(port, username, password, sql)
        if obj and 'model' in obj:
            b = _decode_model_field(obj['model'])
            if b:
                try:
                    sm = torch.jit.load(io.BytesIO(b), map_location='cpu')
                    model.table_histograms[idx].load_state_dict(sm.state_dict())
                    if DEBUG:
                        print(f"DEBUG: loaded histogram {schema}.{table} bytes={len(b)}")
                except Exception as e:
                    print(f"Warning: failed to load histogram {schema}.{table}: {e}")

def exec_sql(port: int, username: str, password: str, sql: str) -> str:
    # Always target the system_statistic database endpoint
    url = SQL_ENDPOINT_TEMPLATE.format(port=port)
    resp = requests.post(url, auth=(username, password), data=sql, timeout=60)
    resp.raise_for_status()
    return resp.text

# Tables are expected to exist; do not create here.

def upload_model_blob(port: int, username: str, password: str, table: str, schema: str, blob_bytes: bytes):
    # Store as base64 string for now (MemCP hex literal unsupported)
    b64 = base64.b64encode(blob_bytes).decode("ascii")
    b64_escaped = b64.replace("'", "''")
    sql = (
        f"INSERT INTO table_histogram(`schema`,`table`,`model`) "
        f"VALUES ('{schema.replace("'","''")}','{table.replace("'","''")}', '{b64_escaped}') "
        f"ON DUPLICATE KEY UPDATE model=VALUES(model)"
    )
    return exec_sql(port, username, password, sql)

def upload_base_model(port: int, username: str, password: str, model_id: str, blob_bytes: bytes):
    b64 = base64.b64encode(blob_bytes).decode("ascii")
    b64_escaped = b64.replace("'", "''")
    sql = (
        f"INSERT INTO base_models(id,model) "
        f"VALUES ('{model_id.replace("'","''")}', '{b64_escaped}') "
        f"ON DUPLICATE KEY UPDATE model=VALUES(model)"
    )
    return exec_sql(port, username, password, sql)

# ---------- Training ----------
def train_model(rows: List[ScanRow], port: int, username: str, password: str, epochs=EPOCHS, reset: bool=False, duration_secs: int | None = None):
    dataset = ScansDataset(rows)
    n_tables = len(dataset.table_keys)
    dataloader = DataLoader(dataset, batch_size=BATCH_SIZE, shuffle=True, collate_fn=collate_fn)
    device = "cpu"
    model = FullTrainingModel(n_tables=n_tables, embed_dim=EMBED_DIM).to(device)
    # Initialize from DB if present unless reset requested
    if not reset:
        try:
            load_existing_models(port, username, password, model, dataset.table_keys)
        except Exception as e:
            print(f"Warning: failed to load existing models: {e}")
    opt = optim.Adam(model.parameters(), lr=LR)
    loss_fn = nn.MSELoss()

    global_step = 0
    start_time = time.time()
    ep = 0
    done = False
    while True:
        ep += 1
        if duration_secs is None and ep > epochs:
            break
        model.train()
        epoch_loss = 0.0
        for batch in dataloader:
            # Build token feature tensors and pad/truncate
            tok_feats = []
            sch_feats = []
            tab_feats = []
            for b in batch:
                feats = token_features(b["tokens"]).to(device)
                if feats.size(0) < MAX_TOKENS:
                    pad = torch.zeros((MAX_TOKENS - feats.size(0), FEATURE_DIM), device=device)
                    feats = torch.cat([feats, pad], dim=0)
                else:
                    feats = feats[:MAX_TOKENS]
                tok_feats.append(feats)
                sch_feats.append(token_features([b["schema"]]).squeeze(0))
                tab_feats.append(token_features([b["table"]]).squeeze(0))

            token_feats = torch.stack(tok_feats, dim=0)
            schema_feats = torch.stack(sch_feats, dim=0)
            table_feats = torch.stack(tab_feats, dim=0)

            table_idx = torch.stack([b["table_idx"] for b in batch]).to(device)
            ordered = torch.stack([b["ordered"] for b in batch]).to(device)
            inputCount = torch.stack([b["inputCount"] for b in batch]).to(device)
            outputCount = torch.stack([b["outputCount"] for b in batch]).to(device)

            target_log = outputCount.log1p()
            target_ratio = outputCount.add(1.0).log() - inputCount.add(1.0).log()

            pred_log, pred_ratio = model(token_feats, schema_feats, table_feats, table_idx, ordered, inputCount)
            loss = loss_fn(pred_log, target_log) + RATIO_LOSS_WEIGHT * loss_fn(pred_ratio, target_ratio)
            opt.zero_grad()
            loss.backward()
            opt.step()

            epoch_loss += loss.item()
            global_step += 1
            if global_step % PRINT_EVERY == 0:
                print(f"Epoch {ep+1} step {global_step} loss {loss.item():.6f}")
            if duration_secs is not None and (time.time() - start_time) >= duration_secs:
                done = True
                break
        print(f"Epoch {ep} completed. Avg loss: {epoch_loss/max(1,len(dataloader)):.6f}")
        if done:
            break

    # --- Export models ---
    base_encoder_script = torch.jit.script(model.base_encoder.cpu())

    # Prepare wrappers, load weights into inner modules, then script
    ord_wrap = OrderedLayerScript(embed_dim=EMBED_DIM).cpu()
    ord_wrap.layer.load_state_dict(model.ordered_layer.state_dict())
    ordered_layer_script = torch.jit.script(ord_wrap)

    out_wrap = OutputHeadScript(embed_dim=EMBED_DIM).cpu()
    out_wrap.head.load_state_dict(model.output_head.state_dict())
    output_head_script = torch.jit.script(out_wrap)

    ratio_mod = RatioHead(embed_dim=EMBED_DIM).cpu()
    ratio_mod.load_state_dict(model.ratio_head.state_dict())
    ratio_head_script = torch.jit.script(ratio_mod)

    # Upload base encoder
    def module_to_bytes(mod: torch.jit.ScriptModule) -> bytes:
        # Always use a real temporary file to avoid accidental file creation
        # with names like "<_io.BytesIO object at ...>".
        import tempfile, os
        with tempfile.NamedTemporaryFile(delete=False) as tf:
            tmp = tf.name
        try:
            torch.jit.save(mod, tmp)
            with open(tmp, 'rb') as f:
                data = f.read()
        finally:
            try:
                os.remove(tmp)
            except Exception:
                pass
        return data

    data = module_to_bytes(base_encoder_script)
    if not data:
        print("Warning: base_encoder_v1 produced empty bytes; skipping upload")
    else:
        upload_base_model(port, username, password, model_id="base_encoder_v1", blob_bytes=data)

    # Upload ordered layer
    data = module_to_bytes(ordered_layer_script)
    if not data:
        print("Warning: ordered_layer_v1 produced empty bytes; skipping upload")
    else:
        upload_base_model(port, username, password, model_id="ordered_layer_v1", blob_bytes=data)

    # Upload output head
    data = module_to_bytes(output_head_script)
    if not data:
        print("Warning: output_head_v1 produced empty bytes; skipping upload")
    else:
        upload_base_model(port, username, password, model_id="output_head_v1", blob_bytes=data)

    # Upload ratio head (auxiliary)
    data = module_to_bytes(ratio_head_script)
    if not data:
        print("Warning: ratio_head_v1 produced empty bytes; skipping upload")
    else:
        upload_base_model(port, username, password, model_id="ratio_head_v1", blob_bytes=data)

    # Upload per-table histogram models
    for idx, (schema, table) in enumerate(dataset.table_keys):
        th = model.table_histograms[idx].cpu()
        th_script = torch.jit.script(th)
        data = module_to_bytes(th_script)
        if not data:
            print(f"Warning: table_histogram {schema}.{table} empty; skipping")
        else:
            upload_model_blob(port, username, password, table=table, schema=schema, blob_bytes=data)

# ---------- Main ----------
def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=DEFAULT_PORT)
    parser.add_argument("--username", type=str, default=DEFAULT_USER)
    parser.add_argument("--password", type=str, default=DEFAULT_PASSWORD)
    parser.add_argument("--reset", action="store_true", help="Ignore existing models; start fresh")
    parser.add_argument("--duration", type=int, default=0, help="Timebox training to N seconds (0=disabled)")
    parser.add_argument("--prompt", action="store_true", help="Interactive prediction loop (no training)")
    parser.add_argument("--ipc", type=str, default="", help="Run shared-memory IPC server at given path")
    parser.add_argument("--debug", action="store_true", help="Verbose debug prints")
    args = parser.parse_args()

    global DEBUG
    DEBUG = bool(args.debug)

    if args.ipc:
        run_ipc_loop(args.ipc, args.port, args.username, args.password)
        return
    if args.prompt:
        run_prompt_loop(args.port, args.username, args.password)
        return

    print("Fetching scan data...")
    rows = fetch_scans(args.port, args.username, args.password)
    if not rows:
        print("No scan rows found, exiting.")
        return

    duration = int(args.duration) if getattr(args, 'duration', 0) else 0
    train_model(rows, args.port, args.username, args.password, epochs=EPOCHS, reset=bool(args.reset), duration_secs=(duration or None))

    # Deployment notes:
    # - This script trains daily and uploads TorchScript models into system_statistic.base_models and system_statistic.table_histogram.
    # - The Go server can pull these blobs at startup or lazily on first use, keep them in-memory, and hot-reload periodically.
    # - Use model IDs with version suffixes (e.g., base_encoder_v1) to enable blue/green rollouts; write both, switch an app-level flag.
    # - Keep an offline shadow path to compare predicted selectivity vs actual and feed deltas back into training data.

    # Go usage sketch (github.com/orktes/go-torch):
    #  baseBytes := fetchBaseModel("base_encoder_v1")
    #  ordBytes  := fetchBaseModel("ordered_layer_v1")
    #  headBytes := fetchBaseModel("output_head_v1")
    #  histBytes := fetchTableHistogram(schema, table)
    #  baseMod, _ := torch.Load(bytes.NewReader(baseBytes))
    #  ordMod,  _ := torch.Load(bytes.NewReader(ordBytes))
    #  headMod, _ := torch.Load(bytes.NewReader(headBytes))
    #  histMod, _ := torch.Load(bytes.NewReader(histBytes))
    #  // Build the same FEATURE_DIM token features in Go and pad to MAX_TOKENS.
    #  tok := torch.FromBlob(tokenFeats, []int64{1, int64(MAX_TOKENS), FEATURE_DIM}, torch.Float32)
    #  sch := torch.FromBlob(schemaFeat, []int64{1, FEATURE_DIM}, torch.Float32)
    #  tab := torch.FromBlob(tableFeat,  []int64{1, FEATURE_DIM}, torch.Float32)
    #  inC := torch.FromBlob([]float32{float32(inputCount)}, []int64{1}, torch.Float32)
    #  baseVec, _ := baseMod.Forward(tok, sch, tab, inC)
    #  histVec, _ := histMod.Forward(baseVec)
    #  if ordered { histVec, _ = ordMod.Forward(histVec) }
    #  logp, _ := headMod.Forward(histVec)
    #  pred := math.Exp(float64(logp.Item().(float32))) - 1.0

if __name__ == "__main__":
    main()
