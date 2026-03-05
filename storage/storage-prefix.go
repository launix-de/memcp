/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import "fmt"
import "strings"
import "github.com/launix-de/memcp/scm"
import "unsafe"

type StoragePrefix struct {
	// prefix compression
	prefixes         StorageInt
	prefixdictionary []string      // pref
	values           StorageString // only one depth (but can be cascaded!)
}

func (s *StoragePrefix) ComputeSize() uint {
	return s.prefixes.ComputeSize() + 24 + s.values.ComputeSize()
}

func (s *StoragePrefix) String() string {
	return fmt.Sprintf("prefix[%s]-%s", s.prefixdictionary[1], s.values.String())
}

func (s *StoragePrefix) GetCachedReader() ColumnReader { return s }

func (s *StoragePrefix) GetValue(i uint32) scm.Scmer {
	inner := s.values.GetValue(i)
	if inner.IsNil() {
		return scm.NewNil()
	}
	if !inner.IsString() {
		panic("invalid value in prefix storage")
	}
	idx := int64(s.prefixes.GetValueUInt(i)) + s.prefixes.offset
	if idx >= int64(len(s.prefixdictionary)) || idx < 0 {
		panic("prefix index out of range")
	}
	prefix := s.prefixdictionary[idx]
	return scm.NewString(prefix + inner.String())
}
func (s *StoragePrefix) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
			var d0 scm.JITValueDesc
			_ = d0
			var d1 scm.JITValueDesc
			_ = d1
			var d2 scm.JITValueDesc
			_ = d2
			var d3 scm.JITValueDesc
			_ = d3
			var r7 unsafe.Pointer
			_ = r7
			var d4 scm.JITValueDesc
			_ = d4
			var d5 scm.JITValueDesc
			_ = d5
			var d6 scm.JITValueDesc
			_ = d6
			var d7 scm.JITValueDesc
			_ = d7
			var d8 scm.JITValueDesc
			_ = d8
			var d9 scm.JITValueDesc
			_ = d9
			var d10 scm.JITValueDesc
			_ = d10
			var d11 scm.JITValueDesc
			_ = d11
			var d12 scm.JITValueDesc
			_ = d12
			var d13 scm.JITValueDesc
			_ = d13
			var d14 scm.JITValueDesc
			_ = d14
			var d15 scm.JITValueDesc
			_ = d15
			var d16 scm.JITValueDesc
			_ = d16
			var d17 scm.JITValueDesc
			_ = d17
			var d18 scm.JITValueDesc
			_ = d18
			var d19 scm.JITValueDesc
			_ = d19
			var d20 scm.JITValueDesc
			_ = d20
			var d21 scm.JITValueDesc
			_ = d21
			var d22 scm.JITValueDesc
			_ = d22
			var d23 scm.JITValueDesc
			_ = d23
			var d24 scm.JITValueDesc
			_ = d24
			var d25 scm.JITValueDesc
			_ = d25
			var d26 scm.JITValueDesc
			_ = d26
			var d27 scm.JITValueDesc
			_ = d27
			var d28 scm.JITValueDesc
			_ = d28
			var d29 scm.JITValueDesc
			_ = d29
			var d30 scm.JITValueDesc
			_ = d30
			var d31 scm.JITValueDesc
			_ = d31
			var d32 scm.JITValueDesc
			_ = d32
			var d33 scm.JITValueDesc
			_ = d33
			var d34 scm.JITValueDesc
			_ = d34
			var d35 scm.JITValueDesc
			_ = d35
			var d36 scm.JITValueDesc
			_ = d36
			var d37 scm.JITValueDesc
			_ = d37
			var d38 scm.JITValueDesc
			_ = d38
			var d39 scm.JITValueDesc
			_ = d39
			var d40 scm.JITValueDesc
			_ = d40
			var d41 scm.JITValueDesc
			_ = d41
			var d42 scm.JITValueDesc
			_ = d42
			var d43 scm.JITValueDesc
			_ = d43
			var d44 scm.JITValueDesc
			_ = d44
			var d45 scm.JITValueDesc
			_ = d45
			var d46 scm.JITValueDesc
			_ = d46
			var d47 scm.JITValueDesc
			_ = d47
			var d48 scm.JITValueDesc
			_ = d48
			var d49 scm.JITValueDesc
			_ = d49
			var d50 scm.JITValueDesc
			_ = d50
			var d51 scm.JITValueDesc
			_ = d51
			var d52 scm.JITValueDesc
			_ = d52
			var d53 scm.JITValueDesc
			_ = d53
			var d54 scm.JITValueDesc
			_ = d54
			var d55 scm.JITValueDesc
			_ = d55
			var d56 scm.JITValueDesc
			_ = d56
			var d57 scm.JITValueDesc
			_ = d57
			var d58 scm.JITValueDesc
			_ = d58
			var d59 scm.JITValueDesc
			_ = d59
			var d60 scm.JITValueDesc
			_ = d60
			var d61 scm.JITValueDesc
			_ = d61
			var d62 scm.JITValueDesc
			_ = d62
			var d63 scm.JITValueDesc
			_ = d63
			var d64 scm.JITValueDesc
			_ = d64
			var d65 scm.JITValueDesc
			_ = d65
			var d66 scm.JITValueDesc
			_ = d66
			var d67 scm.JITValueDesc
			_ = d67
			var d68 scm.JITValueDesc
			_ = d68
			var d69 scm.JITValueDesc
			_ = d69
			var d70 scm.JITValueDesc
			_ = d70
			var d71 scm.JITValueDesc
			_ = d71
			var d72 scm.JITValueDesc
			_ = d72
			var d73 scm.JITValueDesc
			_ = d73
			var d74 scm.JITValueDesc
			_ = d74
			var d75 scm.JITValueDesc
			_ = d75
			var d76 scm.JITValueDesc
			_ = d76
			var d77 scm.JITValueDesc
			_ = d77
			var d78 scm.JITValueDesc
			_ = d78
			var d79 scm.JITValueDesc
			_ = d79
			var d80 scm.JITValueDesc
			_ = d80
			var d81 scm.JITValueDesc
			_ = d81
			var d82 scm.JITValueDesc
			_ = d82
			var d83 scm.JITValueDesc
			_ = d83
			var d84 scm.JITValueDesc
			_ = d84
			var d85 scm.JITValueDesc
			_ = d85
			var d86 scm.JITValueDesc
			_ = d86
			var d87 scm.JITValueDesc
			_ = d87
			var d88 scm.JITValueDesc
			_ = d88
			var d89 scm.JITValueDesc
			_ = d89
			var d90 scm.JITValueDesc
			_ = d90
			var d91 scm.JITValueDesc
			_ = d91
			var d92 scm.JITValueDesc
			_ = d92
			var d93 scm.JITValueDesc
			_ = d93
			var d94 scm.JITValueDesc
			_ = d94
			var d95 scm.JITValueDesc
			_ = d95
			var d96 scm.JITValueDesc
			_ = d96
			var d97 scm.JITValueDesc
			_ = d97
			var d98 scm.JITValueDesc
			_ = d98
			var d99 scm.JITValueDesc
			_ = d99
			var d100 scm.JITValueDesc
			_ = d100
			var d101 scm.JITValueDesc
			_ = d101
			var d102 scm.JITValueDesc
			_ = d102
			var d103 scm.JITValueDesc
			_ = d103
			var d104 scm.JITValueDesc
			_ = d104
			var d105 scm.JITValueDesc
			_ = d105
			var d106 scm.JITValueDesc
			_ = d106
			var d107 scm.JITValueDesc
			_ = d107
			var d108 scm.JITValueDesc
			_ = d108
			var d109 scm.JITValueDesc
			_ = d109
			var d110 scm.JITValueDesc
			_ = d110
			var d111 scm.JITValueDesc
			_ = d111
			var d112 scm.JITValueDesc
			_ = d112
			var d113 scm.JITValueDesc
			_ = d113
			var d114 scm.JITValueDesc
			_ = d114
			var d115 scm.JITValueDesc
			_ = d115
			var d116 scm.JITValueDesc
			_ = d116
			var d117 scm.JITValueDesc
			_ = d117
			var d118 scm.JITValueDesc
			_ = d118
			var d119 scm.JITValueDesc
			_ = d119
			var d120 scm.JITValueDesc
			_ = d120
			var d121 scm.JITValueDesc
			_ = d121
			var d122 scm.JITValueDesc
			_ = d122
			var d123 scm.JITValueDesc
			_ = d123
			var d124 scm.JITValueDesc
			_ = d124
			var d125 scm.JITValueDesc
			_ = d125
			var d126 scm.JITValueDesc
			_ = d126
			var d127 scm.JITValueDesc
			_ = d127
			var d128 scm.JITValueDesc
			_ = d128
			var d129 scm.JITValueDesc
			_ = d129
			var d130 scm.JITValueDesc
			_ = d130
			var d131 scm.JITValueDesc
			_ = d131
			var d132 scm.JITValueDesc
			_ = d132
			var d133 scm.JITValueDesc
			_ = d133
			var d134 scm.JITValueDesc
			_ = d134
			var d135 scm.JITValueDesc
			_ = d135
			var d136 scm.JITValueDesc
			_ = d136
			var d137 scm.JITValueDesc
			_ = d137
			var d138 scm.JITValueDesc
			_ = d138
			var d139 scm.JITValueDesc
			_ = d139
			var d140 scm.JITValueDesc
			_ = d140
			var d141 scm.JITValueDesc
			_ = d141
			var d142 scm.JITValueDesc
			_ = d142
			var d143 scm.JITValueDesc
			_ = d143
			var d144 scm.JITValueDesc
			_ = d144
			var d145 scm.JITValueDesc
			_ = d145
			var d146 scm.JITValueDesc
			_ = d146
			var d147 scm.JITValueDesc
			_ = d147
			var d148 scm.JITValueDesc
			_ = d148
			var d149 scm.JITValueDesc
			_ = d149
			var d150 scm.JITValueDesc
			_ = d150
			var d151 scm.JITValueDesc
			_ = d151
			var d152 scm.JITValueDesc
			_ = d152
			var d153 scm.JITValueDesc
			_ = d153
			var d154 scm.JITValueDesc
			_ = d154
			var d155 scm.JITValueDesc
			_ = d155
			var d156 scm.JITValueDesc
			_ = d156
			var d157 scm.JITValueDesc
			_ = d157
			var d158 scm.JITValueDesc
			_ = d158
			var d159 scm.JITValueDesc
			_ = d159
			var d160 scm.JITValueDesc
			_ = d160
			var d161 scm.JITValueDesc
			_ = d161
			var d162 scm.JITValueDesc
			_ = d162
			var d163 scm.JITValueDesc
			_ = d163
			var d164 scm.JITValueDesc
			_ = d164
			var d165 scm.JITValueDesc
			_ = d165
			var d166 scm.JITValueDesc
			_ = d166
			var d167 scm.JITValueDesc
			_ = d167
			var d168 scm.JITValueDesc
			_ = d168
			var d169 scm.JITValueDesc
			_ = d169
			var d170 scm.JITValueDesc
			_ = d170
			var d171 scm.JITValueDesc
			_ = d171
			var d172 scm.JITValueDesc
			_ = d172
			var d173 scm.JITValueDesc
			_ = d173
			var d174 scm.JITValueDesc
			_ = d174
			var d175 scm.JITValueDesc
			_ = d175
			var d176 scm.JITValueDesc
			_ = d176
			var d177 scm.JITValueDesc
			_ = d177
			var d178 scm.JITValueDesc
			_ = d178
			var d179 scm.JITValueDesc
			_ = d179
			var d180 scm.JITValueDesc
			_ = d180
			var d181 scm.JITValueDesc
			_ = d181
			var d182 scm.JITValueDesc
			_ = d182
			var d183 scm.JITValueDesc
			_ = d183
			var d184 scm.JITValueDesc
			_ = d184
			var d185 scm.JITValueDesc
			_ = d185
			var d186 scm.JITValueDesc
			_ = d186
			var d187 scm.JITValueDesc
			_ = d187
			var d188 scm.JITValueDesc
			_ = d188
			var d189 scm.JITValueDesc
			_ = d189
			var d190 scm.JITValueDesc
			_ = d190
			var d191 scm.JITValueDesc
			_ = d191
			var d192 scm.JITValueDesc
			_ = d192
			var d193 scm.JITValueDesc
			_ = d193
			var d194 scm.JITValueDesc
			_ = d194
			var d195 scm.JITValueDesc
			_ = d195
			var d196 scm.JITValueDesc
			_ = d196
			var d197 scm.JITValueDesc
			_ = d197
			var d198 scm.JITValueDesc
			_ = d198
			var d199 scm.JITValueDesc
			_ = d199
			var d200 scm.JITValueDesc
			_ = d200
			var d201 scm.JITValueDesc
			_ = d201
			var d202 scm.JITValueDesc
			_ = d202
			var d203 scm.JITValueDesc
			_ = d203
			var d204 scm.JITValueDesc
			_ = d204
			var d205 scm.JITValueDesc
			_ = d205
			var d206 scm.JITValueDesc
			_ = d206
			var d207 scm.JITValueDesc
			_ = d207
			var d208 scm.JITValueDesc
			_ = d208
			var d209 scm.JITValueDesc
			_ = d209
			var d210 scm.JITValueDesc
			_ = d210
			var d211 scm.JITValueDesc
			_ = d211
			var d212 scm.JITValueDesc
			_ = d212
			var d213 scm.JITValueDesc
			_ = d213
			var d214 scm.JITValueDesc
			_ = d214
			var d215 scm.JITValueDesc
			_ = d215
			var d216 scm.JITValueDesc
			_ = d216
			var d217 scm.JITValueDesc
			_ = d217
			var d218 scm.JITValueDesc
			_ = d218
			var d219 scm.JITValueDesc
			_ = d219
			var d220 scm.JITValueDesc
			_ = d220
			var d221 scm.JITValueDesc
			_ = d221
			var d222 scm.JITValueDesc
			_ = d222
			var d223 scm.JITValueDesc
			_ = d223
			var d224 scm.JITValueDesc
			_ = d224
			var d225 scm.JITValueDesc
			_ = d225
			var d226 scm.JITValueDesc
			_ = d226
			var d227 scm.JITValueDesc
			_ = d227
			var d228 scm.JITValueDesc
			_ = d228
			var d229 scm.JITValueDesc
			_ = d229
			var d230 scm.JITValueDesc
			_ = d230
			var d231 scm.JITValueDesc
			_ = d231
			var d232 scm.JITValueDesc
			_ = d232
			var d233 scm.JITValueDesc
			_ = d233
			var d234 scm.JITValueDesc
			_ = d234
			var d235 scm.JITValueDesc
			_ = d235
			var d236 scm.JITValueDesc
			_ = d236
			var d237 scm.JITValueDesc
			_ = d237
			var d238 scm.JITValueDesc
			_ = d238
			var d239 scm.JITValueDesc
			_ = d239
			var d240 scm.JITValueDesc
			_ = d240
			var d487 scm.JITValueDesc
			_ = d487
			var d488 scm.JITValueDesc
			_ = d488
			var d489 scm.JITValueDesc
			_ = d489
			var d490 scm.JITValueDesc
			_ = d490
			var d741 scm.JITValueDesc
			_ = d741
			var d742 scm.JITValueDesc
			_ = d742
			var d743 scm.JITValueDesc
			_ = d743
			var d744 scm.JITValueDesc
			_ = d744
			var d745 scm.JITValueDesc
			_ = d745
			var d746 scm.JITValueDesc
			_ = d746
			var d747 scm.JITValueDesc
			_ = d747
			var d748 scm.JITValueDesc
			_ = d748
			var d749 scm.JITValueDesc
			_ = d749
			var d750 scm.JITValueDesc
			_ = d750
			var d751 scm.JITValueDesc
			_ = d751
			var d752 scm.JITValueDesc
			_ = d752
			var d753 scm.JITValueDesc
			_ = d753
			var d754 scm.JITValueDesc
			_ = d754
			var d755 scm.JITValueDesc
			_ = d755
			var d756 scm.JITValueDesc
			_ = d756
			var d757 scm.JITValueDesc
			_ = d757
			var d758 scm.JITValueDesc
			_ = d758
			var d759 scm.JITValueDesc
			_ = d759
			var d760 scm.JITValueDesc
			_ = d760
			var d761 scm.JITValueDesc
			_ = d761
			var d762 scm.JITValueDesc
			_ = d762
			var d763 scm.JITValueDesc
			_ = d763
			var d764 scm.JITValueDesc
			_ = d764
			var d765 scm.JITValueDesc
			_ = d765
			var d766 scm.JITValueDesc
			_ = d766
			var d767 scm.JITValueDesc
			_ = d767
			var d768 scm.JITValueDesc
			_ = d768
			var d769 scm.JITValueDesc
			_ = d769
			var d770 scm.JITValueDesc
			_ = d770
			var d771 scm.JITValueDesc
			_ = d771
			var d772 scm.JITValueDesc
			_ = d772
			var d773 scm.JITValueDesc
			_ = d773
			var d774 scm.JITValueDesc
			_ = d774
			var d775 scm.JITValueDesc
			_ = d775
			var d776 scm.JITValueDesc
			_ = d776
			var d777 scm.JITValueDesc
			_ = d777
			var d778 scm.JITValueDesc
			_ = d778
			var d779 scm.JITValueDesc
			_ = d779
			var d780 scm.JITValueDesc
			_ = d780
			var d781 scm.JITValueDesc
			_ = d781
			var d782 scm.JITValueDesc
			_ = d782
			var d783 scm.JITValueDesc
			_ = d783
			var d784 scm.JITValueDesc
			_ = d784
			var d785 scm.JITValueDesc
			_ = d785
			var d786 scm.JITValueDesc
			_ = d786
			var d787 scm.JITValueDesc
			_ = d787
			var d1085 scm.JITValueDesc
			_ = d1085
			var d1086 scm.JITValueDesc
			_ = d1086
			var d1087 scm.JITValueDesc
			_ = d1087
			var d1088 scm.JITValueDesc
			_ = d1088
			var d1089 scm.JITValueDesc
			_ = d1089
			var d1090 scm.JITValueDesc
			_ = d1090
			var d1091 scm.JITValueDesc
			_ = d1091
			var d1092 scm.JITValueDesc
			_ = d1092
			var d1093 scm.JITValueDesc
			_ = d1093
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var idxInt scm.JITValueDesc
			if idx.Loc == scm.LocImm {
				idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idx.Imm.Int())}
			} else if idx.Loc == scm.LocRegPair {
				ctx.FreeReg(idx.Reg)
				idxInt = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idx.Reg2}
				ctx.BindReg(idx.Reg2, &idxInt)
			} else {
				idxInt = idx
			}
			if idxInt.Loc == scm.LocImm {
				idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(idxInt.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EnsureDesc(&idxInt)
				if idxInt.Loc != scm.LocReg { panic("jit: idxInt not in register") }
				ctx.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			idxPinned := idxInt.Loc == scm.LocReg
			idxPinnedReg := idxInt.Reg
			if idxPinned { ctx.ProtectReg(idxPinnedReg) }
			var bbs [8]scm.BBDescriptor
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			r0 := ctx.AllocReg()
			r1 := ctx.AllocRegExcept(r0)
			lbl0 := ctx.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.ReserveLabel()
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl6 := ctx.ReserveLabel()
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			d0 = idxInt
			_ = d0
			r2 := idxInt.Loc == scm.LocReg
			r3 := idxInt.Reg
			if r2 { ctx.ProtectReg(r3) }
			lbl9 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_4 := int32(-1)
			_ = bbpos_1_4
			bbpos_1_5 := int32(-1)
			_ = bbpos_1_5
			bbpos_1_6 := int32(-1)
			_ = bbpos_1_6
			bbpos_1_7 := int32(-1)
			_ = bbpos_1_7
			bbpos_1_8 := int32(-1)
			_ = bbpos_1_8
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			var d1 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 232
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 232)
				r4 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r4, thisptr.Reg, off)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
				ctx.BindReg(r4, &d1)
			}
			d2 = d1
			ctx.EnsureDesc(&d2)
			if d2.Loc != scm.LocImm && d2.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl10 := ctx.ReserveLabel()
			lbl11 := ctx.ReserveLabel()
			lbl12 := ctx.ReserveLabel()
			lbl13 := ctx.ReserveLabel()
			if d2.Loc == scm.LocImm {
				if d2.Imm.Bool() {
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl10)
				} else {
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl11)
				}
			} else {
				ctx.EmitCmpRegImm32(d2.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl12)
				ctx.EmitJmp(lbl13)
				ctx.MarkLabel(lbl12)
				ctx.EmitJmp(lbl10)
				ctx.MarkLabel(lbl13)
				ctx.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d1)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl11)
			ctx.ResolveFixups()
			ctx.EnsureDesc(&d0)
			d3 = d0
			_ = d3
			r5 := d0.Loc == scm.LocReg
			r6 := d0.Reg
			if r5 { ctx.ProtectReg(r6) }
			r7 = ctx.EmitSubRSP32Fixup()
			_ = r7
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			lbl14 := ctx.ReserveLabel()
			bbpos_2_0 := int32(-1)
			_ = bbpos_2_0
			bbpos_2_1 := int32(-1)
			_ = bbpos_2_1
			bbpos_2_2 := int32(-1)
			_ = bbpos_2_2
			bbpos_2_3 := int32(-1)
			_ = bbpos_2_3
			bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d5 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d3.Imm.Int()))))}
			} else {
				r8 := ctx.AllocReg()
				ctx.EmitMovRegReg(r8, d3.Reg)
				ctx.EmitShlRegImm8(r8, 32)
				ctx.EmitShrRegImm8(r8, 32)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d5)
			}
			var d6 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r9 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r9, thisptr.Reg, off)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
				ctx.BindReg(r9, &d6)
			}
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d6)
			var d7 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d6.Imm.Int()))))}
			} else {
				r10 := ctx.AllocReg()
				ctx.EmitMovRegReg(r10, d6.Reg)
				ctx.EmitShlRegImm8(r10, 56)
				ctx.EmitShrRegImm8(r10, 56)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d7)
			}
			ctx.FreeDesc(&d6)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d7)
			var d8 scm.JITValueDesc
			if d5.Loc == scm.LocImm && d7.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() * d7.Imm.Int())}
			} else if d5.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d7.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d5.Imm.Int()))
				ctx.EmitImulInt64(scratch, d7.Reg)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d8)
			} else if d7.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(scratch, d5.Reg)
				if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d7.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d8)
			} else {
				r11 := ctx.AllocRegExcept(d5.Reg, d7.Reg)
				ctx.EmitMovRegReg(r11, d5.Reg)
				ctx.EmitImulInt64(r11, d7.Reg)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d8)
			}
			if d8.Loc == scm.LocReg && d5.Loc == scm.LocReg && d8.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d5)
			ctx.FreeDesc(&d7)
			var d9 scm.JITValueDesc
			r12 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r12, uint64(dataPtr))
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12, StackOff: int32(sliceLen)}
				ctx.BindReg(r12, &d9)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0)
				ctx.EmitMovRegMem(r12, thisptr.Reg, off)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
				ctx.BindReg(r12, &d9)
			}
			ctx.BindReg(r12, &d9)
			ctx.EnsureDesc(&d8)
			var d10 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() / 64)}
			} else {
				r13 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(r13, d8.Reg)
				ctx.EmitShrRegImm8(r13, 6)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d10)
			}
			if d10.Loc == scm.LocReg && d8.Loc == scm.LocReg && d10.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d10)
			r14 := ctx.AllocReg()
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d9)
			if d10.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r14, uint64(d10.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r14, d10.Reg)
				ctx.EmitShlRegImm8(r14, 3)
			}
			if d9.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitAddInt64(r14, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r14, d9.Reg)
			}
			r15 := ctx.AllocRegExcept(r14)
			ctx.EmitMovRegMem(r15, r14, 0)
			ctx.FreeReg(r14)
			d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			ctx.BindReg(r15, &d11)
			ctx.FreeDesc(&d10)
			ctx.EnsureDesc(&d8)
			var d12 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() % 64)}
			} else {
				r16 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(r16, d8.Reg)
				ctx.EmitAndRegImm32(r16, 63)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d12)
			}
			if d12.Loc == scm.LocReg && d8.Loc == scm.LocReg && d12.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d12)
			var d13 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d12.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d11.Imm.Int()) << uint64(d12.Imm.Int())))}
			} else if d12.Loc == scm.LocImm {
				r17 := ctx.AllocRegExcept(d11.Reg)
				ctx.EmitMovRegReg(r17, d11.Reg)
				ctx.EmitShlRegImm8(r17, uint8(d12.Imm.Int()))
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d13)
			} else {
				{
					shiftSrc := d11.Reg
					r18 := ctx.AllocRegExcept(d11.Reg)
					ctx.EmitMovRegReg(r18, d11.Reg)
					shiftSrc = r18
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d12.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d12.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d12.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d13)
				}
			}
			if d13.Loc == scm.LocReg && d11.Loc == scm.LocReg && d13.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d11)
			ctx.FreeDesc(&d12)
			var d14 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 25)
				r19 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r19, thisptr.Reg, off)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
				ctx.BindReg(r19, &d14)
			}
			d15 = d14
			ctx.EnsureDesc(&d15)
			if d15.Loc != scm.LocImm && d15.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl15 := ctx.ReserveLabel()
			lbl16 := ctx.ReserveLabel()
			lbl17 := ctx.ReserveLabel()
			lbl18 := ctx.ReserveLabel()
			if d15.Loc == scm.LocImm {
				if d15.Imm.Bool() {
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl15)
				} else {
					ctx.MarkLabel(lbl18)
			ctx.EnsureDesc(&d13)
			if d13.Loc == scm.LocReg {
				ctx.ProtectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.ProtectReg(d13.Reg)
				ctx.ProtectReg(d13.Reg2)
			}
			d16 = d13
			if d16.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			ctx.EmitStoreToStack(d16, int32(bbs[2].PhiBase)+int32(0))
			if d13.Loc == scm.LocReg {
				ctx.UnprotectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d13.Reg)
				ctx.UnprotectReg(d13.Reg2)
			}
					ctx.EmitJmp(lbl16)
				}
			} else {
				ctx.EmitCmpRegImm32(d15.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl17)
				ctx.EmitJmp(lbl18)
				ctx.MarkLabel(lbl17)
				ctx.EmitJmp(lbl15)
				ctx.MarkLabel(lbl18)
			ctx.EnsureDesc(&d13)
			if d13.Loc == scm.LocReg {
				ctx.ProtectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.ProtectReg(d13.Reg)
				ctx.ProtectReg(d13.Reg2)
			}
			d17 = d13
			if d17.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d17)
			ctx.EmitStoreToStack(d17, int32(bbs[2].PhiBase)+int32(0))
			if d13.Loc == scm.LocReg {
				ctx.UnprotectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d13.Reg)
				ctx.UnprotectReg(d13.Reg2)
			}
				ctx.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d14)
			bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl16)
			ctx.ResolveFixups()
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			var d18 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r20 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r20, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
				ctx.BindReg(r20, &d18)
			}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			var d19 scm.JITValueDesc
			if d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d18.Imm.Int()))))}
			} else {
				r21 := ctx.AllocReg()
				ctx.EmitMovRegReg(r21, d18.Reg)
				ctx.EmitShlRegImm8(r21, 56)
				ctx.EmitShrRegImm8(r21, 56)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d19)
			}
			ctx.FreeDesc(&d18)
			d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d19)
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm && d19.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() - d19.Imm.Int())}
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				r22 := ctx.AllocRegExcept(d20.Reg)
				ctx.EmitMovRegReg(r22, d20.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d21)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d19.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.EmitSubInt64(scratch, d19.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.EmitMovRegReg(scratch, d20.Reg)
				if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d19.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d19.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else {
				r23 := ctx.AllocRegExcept(d20.Reg, d19.Reg)
				ctx.EmitMovRegReg(r23, d20.Reg)
				ctx.EmitSubInt64(r23, d19.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d21)
			}
			if d21.Loc == scm.LocReg && d20.Loc == scm.LocReg && d21.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d19)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d21)
			var d22 scm.JITValueDesc
			if d4.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d4.Imm.Int()) >> uint64(d21.Imm.Int())))}
			} else if d21.Loc == scm.LocImm {
				r24 := ctx.AllocRegExcept(d4.Reg)
				ctx.EmitMovRegReg(r24, d4.Reg)
				ctx.EmitShrRegImm8(r24, uint8(d21.Imm.Int()))
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d22)
			} else {
				{
					shiftSrc := d4.Reg
					r25 := ctx.AllocRegExcept(d4.Reg)
					ctx.EmitMovRegReg(r25, d4.Reg)
					shiftSrc = r25
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d21.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d21.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d21.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d22)
				}
			}
			if d22.Loc == scm.LocReg && d4.Loc == scm.LocReg && d22.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d4)
			ctx.FreeDesc(&d21)
			r26 := ctx.AllocReg()
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			if d22.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r26, d22)
			}
			ctx.EmitJmp(lbl14)
			bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl15)
			ctx.ResolveFixups()
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d8)
			var d23 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() % 64)}
			} else {
				r27 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(r27, d8.Reg)
				ctx.EmitAndRegImm32(r27, 63)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d23)
			}
			if d23.Loc == scm.LocReg && d8.Loc == scm.LocReg && d23.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			var d24 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r28 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r28, thisptr.Reg, off)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
				ctx.BindReg(r28, &d24)
			}
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d24)
			var d25 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d24.Imm.Int()))))}
			} else {
				r29 := ctx.AllocReg()
				ctx.EmitMovRegReg(r29, d24.Reg)
				ctx.EmitShlRegImm8(r29, 56)
				ctx.EmitShrRegImm8(r29, 56)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d25)
			}
			ctx.FreeDesc(&d24)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			var d26 scm.JITValueDesc
			if d23.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() + d25.Imm.Int())}
			} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
				r30 := ctx.AllocRegExcept(d23.Reg)
				ctx.EmitMovRegReg(r30, d23.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d26)
			} else if d23.Loc == scm.LocImm && d23.Imm.Int() == 0 {
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
				ctx.BindReg(d25.Reg, &d26)
			} else if d23.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d23.Imm.Int()))
				ctx.EmitAddInt64(scratch, d25.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.EmitMovRegReg(scratch, d23.Reg)
				if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d25.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else {
				r31 := ctx.AllocRegExcept(d23.Reg, d25.Reg)
				ctx.EmitMovRegReg(r31, d23.Reg)
				ctx.EmitAddInt64(r31, d25.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d26)
			}
			if d26.Loc == scm.LocReg && d23.Loc == scm.LocReg && d26.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.FreeDesc(&d25)
			ctx.EnsureDesc(&d26)
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d26.Imm.Int()) > uint64(64))}
			} else {
				r32 := ctx.AllocRegExcept(d26.Reg)
				ctx.EmitCmpRegImm32(d26.Reg, 64)
				ctx.EmitSetcc(r32, scm.CcA)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r32}
				ctx.BindReg(r32, &d27)
			}
			ctx.FreeDesc(&d26)
			d28 = d27
			ctx.EnsureDesc(&d28)
			if d28.Loc != scm.LocImm && d28.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl19 := ctx.ReserveLabel()
			lbl20 := ctx.ReserveLabel()
			lbl21 := ctx.ReserveLabel()
			if d28.Loc == scm.LocImm {
				if d28.Imm.Bool() {
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl19)
				} else {
					ctx.MarkLabel(lbl21)
			ctx.EnsureDesc(&d13)
			if d13.Loc == scm.LocReg {
				ctx.ProtectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.ProtectReg(d13.Reg)
				ctx.ProtectReg(d13.Reg2)
			}
			d29 = d13
			if d29.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d29)
			ctx.EmitStoreToStack(d29, int32(bbs[2].PhiBase)+int32(0))
			if d13.Loc == scm.LocReg {
				ctx.UnprotectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d13.Reg)
				ctx.UnprotectReg(d13.Reg2)
			}
					ctx.EmitJmp(lbl16)
				}
			} else {
				ctx.EmitCmpRegImm32(d28.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl20)
				ctx.EmitJmp(lbl21)
				ctx.MarkLabel(lbl20)
				ctx.EmitJmp(lbl19)
				ctx.MarkLabel(lbl21)
			ctx.EnsureDesc(&d13)
			if d13.Loc == scm.LocReg {
				ctx.ProtectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.ProtectReg(d13.Reg)
				ctx.ProtectReg(d13.Reg2)
			}
			d30 = d13
			if d30.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d30)
			ctx.EmitStoreToStack(d30, int32(bbs[2].PhiBase)+int32(0))
			if d13.Loc == scm.LocReg {
				ctx.UnprotectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d13.Reg)
				ctx.UnprotectReg(d13.Reg2)
			}
				ctx.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d27)
			bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl19)
			ctx.ResolveFixups()
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d8)
			var d31 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() / 64)}
			} else {
				r33 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(r33, d8.Reg)
				ctx.EmitShrRegImm8(r33, 6)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d31)
			}
			if d31.Loc == scm.LocReg && d8.Loc == scm.LocReg && d31.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d31)
			var d32 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitMovRegReg(scratch, d31.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d32)
			}
			if d32.Loc == scm.LocReg && d31.Loc == scm.LocReg && d32.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d32)
			r34 := ctx.AllocReg()
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d9)
			if d32.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r34, uint64(d32.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r34, d32.Reg)
				ctx.EmitShlRegImm8(r34, 3)
			}
			if d9.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitAddInt64(r34, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r34, d9.Reg)
			}
			r35 := ctx.AllocRegExcept(r34)
			ctx.EmitMovRegMem(r35, r34, 0)
			ctx.FreeReg(r34)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
			ctx.BindReg(r35, &d33)
			ctx.FreeDesc(&d32)
			ctx.EnsureDesc(&d8)
			var d34 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() % 64)}
			} else {
				r36 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(r36, d8.Reg)
				ctx.EmitAndRegImm32(r36, 63)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d34)
			}
			if d34.Loc == scm.LocReg && d8.Loc == scm.LocReg && d34.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d8)
			d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d34)
			var d36 scm.JITValueDesc
			if d35.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() - d34.Imm.Int())}
			} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
				r37 := ctx.AllocRegExcept(d35.Reg)
				ctx.EmitMovRegReg(r37, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d36)
			} else if d35.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d35.Imm.Int()))
				ctx.EmitSubInt64(scratch, d34.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else if d34.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d35.Reg)
				ctx.EmitMovRegReg(scratch, d35.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d34.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else {
				r38 := ctx.AllocRegExcept(d35.Reg, d34.Reg)
				ctx.EmitMovRegReg(r38, d35.Reg)
				ctx.EmitSubInt64(r38, d34.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d36)
			}
			if d36.Loc == scm.LocReg && d35.Loc == scm.LocReg && d36.Reg == d35.Reg {
				ctx.TransferReg(d35.Reg)
				d35.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d36)
			var d37 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d33.Imm.Int()) >> uint64(d36.Imm.Int())))}
			} else if d36.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d33.Reg)
				ctx.EmitMovRegReg(r39, d33.Reg)
				ctx.EmitShrRegImm8(r39, uint8(d36.Imm.Int()))
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d37)
			} else {
				{
					shiftSrc := d33.Reg
					r40 := ctx.AllocRegExcept(d33.Reg)
					ctx.EmitMovRegReg(r40, d33.Reg)
					shiftSrc = r40
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d36.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d36.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d36.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d37)
				}
			}
			if d37.Loc == scm.LocReg && d33.Loc == scm.LocReg && d37.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d33)
			ctx.FreeDesc(&d36)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d37)
			var d38 scm.JITValueDesc
			if d13.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() | d37.Imm.Int())}
			} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d37.Reg}
				ctx.BindReg(d37.Reg, &d38)
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				r41 := ctx.AllocRegExcept(d13.Reg)
				ctx.EmitMovRegReg(r41, d13.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d38)
			} else if d13.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
				ctx.EmitOrInt64(scratch, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			} else if d37.Loc == scm.LocImm {
				r42 := ctx.AllocRegExcept(d13.Reg)
				ctx.EmitMovRegReg(r42, d13.Reg)
				if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r42, int32(d37.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
					ctx.EmitOrInt64(r42, scm.RegR11)
				}
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d38)
			} else {
				r43 := ctx.AllocRegExcept(d13.Reg, d37.Reg)
				ctx.EmitMovRegReg(r43, d13.Reg)
				ctx.EmitOrInt64(r43, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d38)
			}
			if d38.Loc == scm.LocReg && d13.Loc == scm.LocReg && d38.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			ctx.EnsureDesc(&d38)
			if d38.Loc == scm.LocReg {
				ctx.ProtectReg(d38.Reg)
			} else if d38.Loc == scm.LocRegPair {
				ctx.ProtectReg(d38.Reg)
				ctx.ProtectReg(d38.Reg2)
			}
			d39 = d38
			if d39.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d39)
			ctx.EmitStoreToStack(d39, int32(bbs[2].PhiBase)+int32(0))
			if d38.Loc == scm.LocReg {
				ctx.UnprotectReg(d38.Reg)
			} else if d38.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d38.Reg)
				ctx.UnprotectReg(d38.Reg2)
			}
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl14)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
			ctx.BindReg(r26, &d40)
			ctx.BindReg(r26, &d40)
			if r5 { ctx.UnprotectReg(r6) }
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			var d41 scm.JITValueDesc
			if d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d40.Imm.Int()))))}
			} else {
				r44 := ctx.AllocReg()
				ctx.EmitMovRegReg(r44, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d41)
			}
			ctx.FreeDesc(&d40)
			var d42 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32)
				r45 := ctx.AllocReg()
				ctx.EmitMovRegMem(r45, thisptr.Reg, off)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d42)
			}
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d42)
			var d43 scm.JITValueDesc
			if d41.Loc == scm.LocImm && d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() + d42.Imm.Int())}
			} else if d42.Loc == scm.LocImm && d42.Imm.Int() == 0 {
				r46 := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegReg(r46, d41.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d43)
			} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d42.Reg}
				ctx.BindReg(d42.Reg, &d43)
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d42.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d41.Imm.Int()))
				ctx.EmitAddInt64(scratch, d42.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else if d42.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegReg(scratch, d41.Reg)
				if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d42.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else {
				r47 := ctx.AllocRegExcept(d41.Reg, d42.Reg)
				ctx.EmitMovRegReg(r47, d41.Reg)
				ctx.EmitAddInt64(r47, d42.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d43)
			}
			if d43.Loc == scm.LocReg && d41.Loc == scm.LocReg && d43.Reg == d41.Reg {
				ctx.TransferReg(d41.Reg)
				d41.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d41)
			ctx.FreeDesc(&d42)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d43)
			var d44 scm.JITValueDesc
			if d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d43.Imm.Int()))))}
			} else {
				r48 := ctx.AllocReg()
				ctx.EmitMovRegReg(r48, d43.Reg)
				ctx.EmitShlRegImm8(r48, 32)
				ctx.EmitShrRegImm8(r48, 32)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d44)
			}
			ctx.FreeDesc(&d43)
			var d45 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56)
				r49 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r49, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
				ctx.BindReg(r49, &d45)
			}
			d46 = d45
			ctx.EnsureDesc(&d46)
			if d46.Loc != scm.LocImm && d46.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl22 := ctx.ReserveLabel()
			lbl23 := ctx.ReserveLabel()
			lbl24 := ctx.ReserveLabel()
			lbl25 := ctx.ReserveLabel()
			if d46.Loc == scm.LocImm {
				if d46.Imm.Bool() {
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl22)
				} else {
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl23)
				}
			} else {
				ctx.EmitCmpRegImm32(d46.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl24)
				ctx.EmitJmp(lbl25)
				ctx.MarkLabel(lbl24)
				ctx.EmitJmp(lbl22)
				ctx.MarkLabel(lbl25)
				ctx.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d45)
			bbpos_1_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl23)
			ctx.ResolveFixups()
			ctx.EnsureDesc(&d44)
			d47 = d44
			_ = d47
			r50 := d44.Loc == scm.LocReg
			r51 := d44.Reg
			if r50 { ctx.ProtectReg(r51) }
			d48 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			lbl26 := ctx.ReserveLabel()
			bbpos_3_0 := int32(-1)
			_ = bbpos_3_0
			bbpos_3_1 := int32(-1)
			_ = bbpos_3_1
			bbpos_3_2 := int32(-1)
			_ = bbpos_3_2
			bbpos_3_3 := int32(-1)
			_ = bbpos_3_3
			bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d48 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d47)
			var d49 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d47.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.EmitMovRegReg(r52, d47.Reg)
				ctx.EmitShlRegImm8(r52, 32)
				ctx.EmitShrRegImm8(r52, 32)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d49)
			}
			var d50 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r53 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r53, thisptr.Reg, off)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r53}
				ctx.BindReg(r53, &d50)
			}
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d50)
			var d51 scm.JITValueDesc
			if d50.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d50.Imm.Int()))))}
			} else {
				r54 := ctx.AllocReg()
				ctx.EmitMovRegReg(r54, d50.Reg)
				ctx.EmitShlRegImm8(r54, 56)
				ctx.EmitShrRegImm8(r54, 56)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d51)
			}
			ctx.FreeDesc(&d50)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d51)
			var d52 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d51.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() * d51.Imm.Int())}
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d51.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
				ctx.EmitImulInt64(scratch, d51.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else if d51.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.EmitMovRegReg(scratch, d49.Reg)
				if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d51.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else {
				r55 := ctx.AllocRegExcept(d49.Reg, d51.Reg)
				ctx.EmitMovRegReg(r55, d49.Reg)
				ctx.EmitImulInt64(r55, d51.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d52)
			}
			if d52.Loc == scm.LocReg && d49.Loc == scm.LocReg && d52.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d49)
			ctx.FreeDesc(&d51)
			var d53 scm.JITValueDesc
			r56 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r56, uint64(dataPtr))
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56, StackOff: int32(sliceLen)}
				ctx.BindReg(r56, &d53)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.EmitMovRegMem(r56, thisptr.Reg, off)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
				ctx.BindReg(r56, &d53)
			}
			ctx.BindReg(r56, &d53)
			ctx.EnsureDesc(&d52)
			var d54 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() / 64)}
			} else {
				r57 := ctx.AllocRegExcept(d52.Reg)
				ctx.EmitMovRegReg(r57, d52.Reg)
				ctx.EmitShrRegImm8(r57, 6)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
				ctx.BindReg(r57, &d54)
			}
			if d54.Loc == scm.LocReg && d52.Loc == scm.LocReg && d54.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d54)
			r58 := ctx.AllocReg()
			ctx.EnsureDesc(&d54)
			ctx.EnsureDesc(&d53)
			if d54.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r58, uint64(d54.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r58, d54.Reg)
				ctx.EmitShlRegImm8(r58, 3)
			}
			if d53.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.EmitAddInt64(r58, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r58, d53.Reg)
			}
			r59 := ctx.AllocRegExcept(r58)
			ctx.EmitMovRegMem(r59, r58, 0)
			ctx.FreeReg(r58)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
			ctx.BindReg(r59, &d55)
			ctx.FreeDesc(&d54)
			ctx.EnsureDesc(&d52)
			var d56 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() % 64)}
			} else {
				r60 := ctx.AllocRegExcept(d52.Reg)
				ctx.EmitMovRegReg(r60, d52.Reg)
				ctx.EmitAndRegImm32(r60, 63)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
				ctx.BindReg(r60, &d56)
			}
			if d56.Loc == scm.LocReg && d52.Loc == scm.LocReg && d56.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d56)
			var d57 scm.JITValueDesc
			if d55.Loc == scm.LocImm && d56.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d55.Imm.Int()) << uint64(d56.Imm.Int())))}
			} else if d56.Loc == scm.LocImm {
				r61 := ctx.AllocRegExcept(d55.Reg)
				ctx.EmitMovRegReg(r61, d55.Reg)
				ctx.EmitShlRegImm8(r61, uint8(d56.Imm.Int()))
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
				ctx.BindReg(r61, &d57)
			} else {
				{
					shiftSrc := d55.Reg
					r62 := ctx.AllocRegExcept(d55.Reg)
					ctx.EmitMovRegReg(r62, d55.Reg)
					shiftSrc = r62
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d56.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d56.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d56.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d57)
				}
			}
			if d57.Loc == scm.LocReg && d55.Loc == scm.LocReg && d57.Reg == d55.Reg {
				ctx.TransferReg(d55.Reg)
				d55.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d55)
			ctx.FreeDesc(&d56)
			var d58 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r63 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r63, thisptr.Reg, off)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
				ctx.BindReg(r63, &d58)
			}
			d59 = d58
			ctx.EnsureDesc(&d59)
			if d59.Loc != scm.LocImm && d59.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl27 := ctx.ReserveLabel()
			lbl28 := ctx.ReserveLabel()
			lbl29 := ctx.ReserveLabel()
			lbl30 := ctx.ReserveLabel()
			if d59.Loc == scm.LocImm {
				if d59.Imm.Bool() {
					ctx.MarkLabel(lbl29)
					ctx.EmitJmp(lbl27)
				} else {
					ctx.MarkLabel(lbl30)
			ctx.EnsureDesc(&d57)
			if d57.Loc == scm.LocReg {
				ctx.ProtectReg(d57.Reg)
			} else if d57.Loc == scm.LocRegPair {
				ctx.ProtectReg(d57.Reg)
				ctx.ProtectReg(d57.Reg2)
			}
			d60 = d57
			if d60.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d60)
			ctx.EmitStoreToStack(d60, int32(bbs[2].PhiBase)+int32(0))
			if d57.Loc == scm.LocReg {
				ctx.UnprotectReg(d57.Reg)
			} else if d57.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d57.Reg)
				ctx.UnprotectReg(d57.Reg2)
			}
					ctx.EmitJmp(lbl28)
				}
			} else {
				ctx.EmitCmpRegImm32(d59.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl29)
				ctx.EmitJmp(lbl30)
				ctx.MarkLabel(lbl29)
				ctx.EmitJmp(lbl27)
				ctx.MarkLabel(lbl30)
			ctx.EnsureDesc(&d57)
			if d57.Loc == scm.LocReg {
				ctx.ProtectReg(d57.Reg)
			} else if d57.Loc == scm.LocRegPair {
				ctx.ProtectReg(d57.Reg)
				ctx.ProtectReg(d57.Reg2)
			}
			d61 = d57
			if d61.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d61)
			ctx.EmitStoreToStack(d61, int32(bbs[2].PhiBase)+int32(0))
			if d57.Loc == scm.LocReg {
				ctx.UnprotectReg(d57.Reg)
			} else if d57.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d57.Reg)
				ctx.UnprotectReg(d57.Reg2)
			}
				ctx.EmitJmp(lbl28)
			}
			ctx.FreeDesc(&d58)
			bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl28)
			ctx.ResolveFixups()
			d48 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			var d62 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r64 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r64, thisptr.Reg, off)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
				ctx.BindReg(r64, &d62)
			}
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d62)
			var d63 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d62.Imm.Int()))))}
			} else {
				r65 := ctx.AllocReg()
				ctx.EmitMovRegReg(r65, d62.Reg)
				ctx.EmitShlRegImm8(r65, 56)
				ctx.EmitShrRegImm8(r65, 56)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d63)
			}
			ctx.FreeDesc(&d62)
			d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d63)
			var d65 scm.JITValueDesc
			if d64.Loc == scm.LocImm && d63.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d64.Imm.Int() - d63.Imm.Int())}
			} else if d63.Loc == scm.LocImm && d63.Imm.Int() == 0 {
				r66 := ctx.AllocRegExcept(d64.Reg)
				ctx.EmitMovRegReg(r66, d64.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d65)
			} else if d64.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d63.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d64.Imm.Int()))
				ctx.EmitSubInt64(scratch, d63.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			} else if d63.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d64.Reg)
				ctx.EmitMovRegReg(scratch, d64.Reg)
				if d63.Imm.Int() >= -2147483648 && d63.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d63.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			} else {
				r67 := ctx.AllocRegExcept(d64.Reg, d63.Reg)
				ctx.EmitMovRegReg(r67, d64.Reg)
				ctx.EmitSubInt64(r67, d63.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d65)
			}
			if d65.Loc == scm.LocReg && d64.Loc == scm.LocReg && d65.Reg == d64.Reg {
				ctx.TransferReg(d64.Reg)
				d64.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d63)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d65)
			var d66 scm.JITValueDesc
			if d48.Loc == scm.LocImm && d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d48.Imm.Int()) >> uint64(d65.Imm.Int())))}
			} else if d65.Loc == scm.LocImm {
				r68 := ctx.AllocRegExcept(d48.Reg)
				ctx.EmitMovRegReg(r68, d48.Reg)
				ctx.EmitShrRegImm8(r68, uint8(d65.Imm.Int()))
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
				ctx.BindReg(r68, &d66)
			} else {
				{
					shiftSrc := d48.Reg
					r69 := ctx.AllocRegExcept(d48.Reg)
					ctx.EmitMovRegReg(r69, d48.Reg)
					shiftSrc = r69
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d65.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d65.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d65.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d66)
				}
			}
			if d66.Loc == scm.LocReg && d48.Loc == scm.LocReg && d66.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
			ctx.FreeDesc(&d65)
			r70 := ctx.AllocReg()
			ctx.EnsureDesc(&d66)
			ctx.EnsureDesc(&d66)
			if d66.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r70, d66)
			}
			ctx.EmitJmp(lbl26)
			bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl27)
			ctx.ResolveFixups()
			d48 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			ctx.EnsureDesc(&d52)
			var d67 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() % 64)}
			} else {
				r71 := ctx.AllocRegExcept(d52.Reg)
				ctx.EmitMovRegReg(r71, d52.Reg)
				ctx.EmitAndRegImm32(r71, 63)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d67)
			}
			if d67.Loc == scm.LocReg && d52.Loc == scm.LocReg && d67.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			var d68 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r72 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r72, thisptr.Reg, off)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
				ctx.BindReg(r72, &d68)
			}
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d68)
			var d69 scm.JITValueDesc
			if d68.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d68.Imm.Int()))))}
			} else {
				r73 := ctx.AllocReg()
				ctx.EmitMovRegReg(r73, d68.Reg)
				ctx.EmitShlRegImm8(r73, 56)
				ctx.EmitShrRegImm8(r73, 56)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d69)
			}
			ctx.FreeDesc(&d68)
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d69)
			var d70 scm.JITValueDesc
			if d67.Loc == scm.LocImm && d69.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d67.Imm.Int() + d69.Imm.Int())}
			} else if d69.Loc == scm.LocImm && d69.Imm.Int() == 0 {
				r74 := ctx.AllocRegExcept(d67.Reg)
				ctx.EmitMovRegReg(r74, d67.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
				ctx.BindReg(r74, &d70)
			} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d69.Reg}
				ctx.BindReg(d69.Reg, &d70)
			} else if d67.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d69.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d67.Imm.Int()))
				ctx.EmitAddInt64(scratch, d69.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d70)
			} else if d69.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d67.Reg)
				ctx.EmitMovRegReg(scratch, d67.Reg)
				if d69.Imm.Int() >= -2147483648 && d69.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d69.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d69.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d70)
			} else {
				r75 := ctx.AllocRegExcept(d67.Reg, d69.Reg)
				ctx.EmitMovRegReg(r75, d67.Reg)
				ctx.EmitAddInt64(r75, d69.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d70)
			}
			if d70.Loc == scm.LocReg && d67.Loc == scm.LocReg && d70.Reg == d67.Reg {
				ctx.TransferReg(d67.Reg)
				d67.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d67)
			ctx.FreeDesc(&d69)
			ctx.EnsureDesc(&d70)
			var d71 scm.JITValueDesc
			if d70.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d70.Imm.Int()) > uint64(64))}
			} else {
				r76 := ctx.AllocRegExcept(d70.Reg)
				ctx.EmitCmpRegImm32(d70.Reg, 64)
				ctx.EmitSetcc(r76, scm.CcA)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r76}
				ctx.BindReg(r76, &d71)
			}
			ctx.FreeDesc(&d70)
			d72 = d71
			ctx.EnsureDesc(&d72)
			if d72.Loc != scm.LocImm && d72.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl31 := ctx.ReserveLabel()
			lbl32 := ctx.ReserveLabel()
			lbl33 := ctx.ReserveLabel()
			if d72.Loc == scm.LocImm {
				if d72.Imm.Bool() {
					ctx.MarkLabel(lbl32)
					ctx.EmitJmp(lbl31)
				} else {
					ctx.MarkLabel(lbl33)
			ctx.EnsureDesc(&d57)
			if d57.Loc == scm.LocReg {
				ctx.ProtectReg(d57.Reg)
			} else if d57.Loc == scm.LocRegPair {
				ctx.ProtectReg(d57.Reg)
				ctx.ProtectReg(d57.Reg2)
			}
			d73 = d57
			if d73.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d73)
			ctx.EmitStoreToStack(d73, int32(bbs[2].PhiBase)+int32(0))
			if d57.Loc == scm.LocReg {
				ctx.UnprotectReg(d57.Reg)
			} else if d57.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d57.Reg)
				ctx.UnprotectReg(d57.Reg2)
			}
					ctx.EmitJmp(lbl28)
				}
			} else {
				ctx.EmitCmpRegImm32(d72.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl32)
				ctx.EmitJmp(lbl33)
				ctx.MarkLabel(lbl32)
				ctx.EmitJmp(lbl31)
				ctx.MarkLabel(lbl33)
			ctx.EnsureDesc(&d57)
			if d57.Loc == scm.LocReg {
				ctx.ProtectReg(d57.Reg)
			} else if d57.Loc == scm.LocRegPair {
				ctx.ProtectReg(d57.Reg)
				ctx.ProtectReg(d57.Reg2)
			}
			d74 = d57
			if d74.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d74)
			ctx.EmitStoreToStack(d74, int32(bbs[2].PhiBase)+int32(0))
			if d57.Loc == scm.LocReg {
				ctx.UnprotectReg(d57.Reg)
			} else if d57.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d57.Reg)
				ctx.UnprotectReg(d57.Reg2)
			}
				ctx.EmitJmp(lbl28)
			}
			ctx.FreeDesc(&d71)
			bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl31)
			ctx.ResolveFixups()
			d48 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			ctx.EnsureDesc(&d52)
			var d75 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() / 64)}
			} else {
				r77 := ctx.AllocRegExcept(d52.Reg)
				ctx.EmitMovRegReg(r77, d52.Reg)
				ctx.EmitShrRegImm8(r77, 6)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
				ctx.BindReg(r77, &d75)
			}
			if d75.Loc == scm.LocReg && d52.Loc == scm.LocReg && d75.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d75)
			var d76 scm.JITValueDesc
			if d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d75.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d75.Reg)
				ctx.EmitMovRegReg(scratch, d75.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d76)
			}
			if d76.Loc == scm.LocReg && d75.Loc == scm.LocReg && d76.Reg == d75.Reg {
				ctx.TransferReg(d75.Reg)
				d75.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d75)
			ctx.EnsureDesc(&d76)
			r78 := ctx.AllocReg()
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d53)
			if d76.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r78, uint64(d76.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r78, d76.Reg)
				ctx.EmitShlRegImm8(r78, 3)
			}
			if d53.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.EmitAddInt64(r78, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r78, d53.Reg)
			}
			r79 := ctx.AllocRegExcept(r78)
			ctx.EmitMovRegMem(r79, r78, 0)
			ctx.FreeReg(r78)
			d77 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
			ctx.BindReg(r79, &d77)
			ctx.FreeDesc(&d76)
			ctx.EnsureDesc(&d52)
			var d78 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() % 64)}
			} else {
				r80 := ctx.AllocRegExcept(d52.Reg)
				ctx.EmitMovRegReg(r80, d52.Reg)
				ctx.EmitAndRegImm32(r80, 63)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d78)
			}
			if d78.Loc == scm.LocReg && d52.Loc == scm.LocReg && d78.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d52)
			d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d78)
			var d80 scm.JITValueDesc
			if d79.Loc == scm.LocImm && d78.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d79.Imm.Int() - d78.Imm.Int())}
			} else if d78.Loc == scm.LocImm && d78.Imm.Int() == 0 {
				r81 := ctx.AllocRegExcept(d79.Reg)
				ctx.EmitMovRegReg(r81, d79.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d80)
			} else if d79.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d78.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d79.Imm.Int()))
				ctx.EmitSubInt64(scratch, d78.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else if d78.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d79.Reg)
				ctx.EmitMovRegReg(scratch, d79.Reg)
				if d78.Imm.Int() >= -2147483648 && d78.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d78.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d78.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else {
				r82 := ctx.AllocRegExcept(d79.Reg, d78.Reg)
				ctx.EmitMovRegReg(r82, d79.Reg)
				ctx.EmitSubInt64(r82, d78.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d80)
			}
			if d80.Loc == scm.LocReg && d79.Loc == scm.LocReg && d80.Reg == d79.Reg {
				ctx.TransferReg(d79.Reg)
				d79.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d78)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d80)
			var d81 scm.JITValueDesc
			if d77.Loc == scm.LocImm && d80.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d77.Imm.Int()) >> uint64(d80.Imm.Int())))}
			} else if d80.Loc == scm.LocImm {
				r83 := ctx.AllocRegExcept(d77.Reg)
				ctx.EmitMovRegReg(r83, d77.Reg)
				ctx.EmitShrRegImm8(r83, uint8(d80.Imm.Int()))
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d81)
			} else {
				{
					shiftSrc := d77.Reg
					r84 := ctx.AllocRegExcept(d77.Reg)
					ctx.EmitMovRegReg(r84, d77.Reg)
					shiftSrc = r84
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d80.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d80.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d80.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d81)
				}
			}
			if d81.Loc == scm.LocReg && d77.Loc == scm.LocReg && d81.Reg == d77.Reg {
				ctx.TransferReg(d77.Reg)
				d77.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d77)
			ctx.FreeDesc(&d80)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d81)
			var d82 scm.JITValueDesc
			if d57.Loc == scm.LocImm && d81.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() | d81.Imm.Int())}
			} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d81.Reg}
				ctx.BindReg(d81.Reg, &d82)
			} else if d81.Loc == scm.LocImm && d81.Imm.Int() == 0 {
				r85 := ctx.AllocRegExcept(d57.Reg)
				ctx.EmitMovRegReg(r85, d57.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d82)
			} else if d57.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d81.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d57.Imm.Int()))
				ctx.EmitOrInt64(scratch, d81.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d82)
			} else if d81.Loc == scm.LocImm {
				r86 := ctx.AllocRegExcept(d57.Reg)
				ctx.EmitMovRegReg(r86, d57.Reg)
				if d81.Imm.Int() >= -2147483648 && d81.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r86, int32(d81.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d81.Imm.Int()))
					ctx.EmitOrInt64(r86, scm.RegR11)
				}
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d82)
			} else {
				r87 := ctx.AllocRegExcept(d57.Reg, d81.Reg)
				ctx.EmitMovRegReg(r87, d57.Reg)
				ctx.EmitOrInt64(r87, d81.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d82)
			}
			if d82.Loc == scm.LocReg && d57.Loc == scm.LocReg && d82.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d81)
			ctx.EnsureDesc(&d82)
			if d82.Loc == scm.LocReg {
				ctx.ProtectReg(d82.Reg)
			} else if d82.Loc == scm.LocRegPair {
				ctx.ProtectReg(d82.Reg)
				ctx.ProtectReg(d82.Reg2)
			}
			d83 = d82
			if d83.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d83)
			ctx.EmitStoreToStack(d83, int32(bbs[2].PhiBase)+int32(0))
			if d82.Loc == scm.LocReg {
				ctx.UnprotectReg(d82.Reg)
			} else if d82.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d82.Reg)
				ctx.UnprotectReg(d82.Reg2)
			}
			ctx.EmitJmp(lbl28)
			ctx.MarkLabel(lbl26)
			d84 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
			ctx.BindReg(r70, &d84)
			ctx.BindReg(r70, &d84)
			if r50 { ctx.UnprotectReg(r51) }
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d84)
			var d85 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d84.Imm.Int()))))}
			} else {
				r88 := ctx.AllocReg()
				ctx.EmitMovRegReg(r88, d84.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d85)
			}
			ctx.FreeDesc(&d84)
			var d86 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r89 := ctx.AllocReg()
				ctx.EmitMovRegMem(r89, thisptr.Reg, off)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r89}
				ctx.BindReg(r89, &d86)
			}
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d86)
			var d87 scm.JITValueDesc
			if d85.Loc == scm.LocImm && d86.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d85.Imm.Int() + d86.Imm.Int())}
			} else if d86.Loc == scm.LocImm && d86.Imm.Int() == 0 {
				r90 := ctx.AllocRegExcept(d85.Reg)
				ctx.EmitMovRegReg(r90, d85.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d87)
			} else if d85.Loc == scm.LocImm && d85.Imm.Int() == 0 {
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
				ctx.BindReg(d86.Reg, &d87)
			} else if d85.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d86.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d85.Imm.Int()))
				ctx.EmitAddInt64(scratch, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d87)
			} else if d86.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d85.Reg)
				ctx.EmitMovRegReg(scratch, d85.Reg)
				if d86.Imm.Int() >= -2147483648 && d86.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d86.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d86.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d87)
			} else {
				r91 := ctx.AllocRegExcept(d85.Reg, d86.Reg)
				ctx.EmitMovRegReg(r91, d85.Reg)
				ctx.EmitAddInt64(r91, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
				ctx.BindReg(r91, &d87)
			}
			if d87.Loc == scm.LocReg && d85.Loc == scm.LocReg && d87.Reg == d85.Reg {
				ctx.TransferReg(d85.Reg)
				d85.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d85)
			ctx.FreeDesc(&d86)
			ctx.EnsureDesc(&d44)
			d88 = d44
			_ = d88
			r92 := d44.Loc == scm.LocReg
			r93 := d44.Reg
			if r92 { ctx.ProtectReg(r93) }
			d89 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			lbl34 := ctx.ReserveLabel()
			bbpos_4_0 := int32(-1)
			_ = bbpos_4_0
			bbpos_4_1 := int32(-1)
			_ = bbpos_4_1
			bbpos_4_2 := int32(-1)
			_ = bbpos_4_2
			bbpos_4_3 := int32(-1)
			_ = bbpos_4_3
			bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d89 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d88)
			var d90 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d88.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.EmitMovRegReg(r94, d88.Reg)
				ctx.EmitShlRegImm8(r94, 32)
				ctx.EmitShrRegImm8(r94, 32)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d90)
			}
			var d91 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r95 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r95, thisptr.Reg, off)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
				ctx.BindReg(r95, &d91)
			}
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d91)
			var d92 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d91.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.EmitMovRegReg(r96, d91.Reg)
				ctx.EmitShlRegImm8(r96, 56)
				ctx.EmitShrRegImm8(r96, 56)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d92)
			}
			ctx.FreeDesc(&d91)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d92)
			var d93 scm.JITValueDesc
			if d90.Loc == scm.LocImm && d92.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d90.Imm.Int() * d92.Imm.Int())}
			} else if d90.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d92.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d90.Imm.Int()))
				ctx.EmitImulInt64(scratch, d92.Reg)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d93)
			} else if d92.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d90.Reg)
				ctx.EmitMovRegReg(scratch, d90.Reg)
				if d92.Imm.Int() >= -2147483648 && d92.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d92.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d92.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d93)
			} else {
				r97 := ctx.AllocRegExcept(d90.Reg, d92.Reg)
				ctx.EmitMovRegReg(r97, d90.Reg)
				ctx.EmitImulInt64(r97, d92.Reg)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d93)
			}
			if d93.Loc == scm.LocReg && d90.Loc == scm.LocReg && d93.Reg == d90.Reg {
				ctx.TransferReg(d90.Reg)
				d90.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d90)
			ctx.FreeDesc(&d92)
			var d94 scm.JITValueDesc
			r98 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r98, uint64(dataPtr))
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98, StackOff: int32(sliceLen)}
				ctx.BindReg(r98, &d94)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.EmitMovRegMem(r98, thisptr.Reg, off)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
				ctx.BindReg(r98, &d94)
			}
			ctx.BindReg(r98, &d94)
			ctx.EnsureDesc(&d93)
			var d95 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() / 64)}
			} else {
				r99 := ctx.AllocRegExcept(d93.Reg)
				ctx.EmitMovRegReg(r99, d93.Reg)
				ctx.EmitShrRegImm8(r99, 6)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d95)
			}
			if d95.Loc == scm.LocReg && d93.Loc == scm.LocReg && d95.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d95)
			r100 := ctx.AllocReg()
			ctx.EnsureDesc(&d95)
			ctx.EnsureDesc(&d94)
			if d95.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r100, uint64(d95.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r100, d95.Reg)
				ctx.EmitShlRegImm8(r100, 3)
			}
			if d94.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d94.Imm.Int()))
				ctx.EmitAddInt64(r100, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r100, d94.Reg)
			}
			r101 := ctx.AllocRegExcept(r100)
			ctx.EmitMovRegMem(r101, r100, 0)
			ctx.FreeReg(r100)
			d96 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
			ctx.BindReg(r101, &d96)
			ctx.FreeDesc(&d95)
			ctx.EnsureDesc(&d93)
			var d97 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() % 64)}
			} else {
				r102 := ctx.AllocRegExcept(d93.Reg)
				ctx.EmitMovRegReg(r102, d93.Reg)
				ctx.EmitAndRegImm32(r102, 63)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d97)
			}
			if d97.Loc == scm.LocReg && d93.Loc == scm.LocReg && d97.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d96)
			ctx.EnsureDesc(&d97)
			var d98 scm.JITValueDesc
			if d96.Loc == scm.LocImm && d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d96.Imm.Int()) << uint64(d97.Imm.Int())))}
			} else if d97.Loc == scm.LocImm {
				r103 := ctx.AllocRegExcept(d96.Reg)
				ctx.EmitMovRegReg(r103, d96.Reg)
				ctx.EmitShlRegImm8(r103, uint8(d97.Imm.Int()))
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d98)
			} else {
				{
					shiftSrc := d96.Reg
					r104 := ctx.AllocRegExcept(d96.Reg)
					ctx.EmitMovRegReg(r104, d96.Reg)
					shiftSrc = r104
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d97.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d97.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d97.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d98)
				}
			}
			if d98.Loc == scm.LocReg && d96.Loc == scm.LocReg && d98.Reg == d96.Reg {
				ctx.TransferReg(d96.Reg)
				d96.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d96)
			ctx.FreeDesc(&d97)
			var d99 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r105 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r105, thisptr.Reg, off)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
				ctx.BindReg(r105, &d99)
			}
			d100 = d99
			ctx.EnsureDesc(&d100)
			if d100.Loc != scm.LocImm && d100.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl35 := ctx.ReserveLabel()
			lbl36 := ctx.ReserveLabel()
			lbl37 := ctx.ReserveLabel()
			lbl38 := ctx.ReserveLabel()
			if d100.Loc == scm.LocImm {
				if d100.Imm.Bool() {
					ctx.MarkLabel(lbl37)
					ctx.EmitJmp(lbl35)
				} else {
					ctx.MarkLabel(lbl38)
			ctx.EnsureDesc(&d98)
			if d98.Loc == scm.LocReg {
				ctx.ProtectReg(d98.Reg)
			} else if d98.Loc == scm.LocRegPair {
				ctx.ProtectReg(d98.Reg)
				ctx.ProtectReg(d98.Reg2)
			}
			d101 = d98
			if d101.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d101)
			ctx.EmitStoreToStack(d101, int32(bbs[2].PhiBase)+int32(0))
			if d98.Loc == scm.LocReg {
				ctx.UnprotectReg(d98.Reg)
			} else if d98.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d98.Reg)
				ctx.UnprotectReg(d98.Reg2)
			}
					ctx.EmitJmp(lbl36)
				}
			} else {
				ctx.EmitCmpRegImm32(d100.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl37)
				ctx.EmitJmp(lbl38)
				ctx.MarkLabel(lbl37)
				ctx.EmitJmp(lbl35)
				ctx.MarkLabel(lbl38)
			ctx.EnsureDesc(&d98)
			if d98.Loc == scm.LocReg {
				ctx.ProtectReg(d98.Reg)
			} else if d98.Loc == scm.LocRegPair {
				ctx.ProtectReg(d98.Reg)
				ctx.ProtectReg(d98.Reg2)
			}
			d102 = d98
			if d102.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d102)
			ctx.EmitStoreToStack(d102, int32(bbs[2].PhiBase)+int32(0))
			if d98.Loc == scm.LocReg {
				ctx.UnprotectReg(d98.Reg)
			} else if d98.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d98.Reg)
				ctx.UnprotectReg(d98.Reg2)
			}
				ctx.EmitJmp(lbl36)
			}
			ctx.FreeDesc(&d99)
			bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl36)
			ctx.ResolveFixups()
			d89 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			var d103 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r106 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r106, thisptr.Reg, off)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r106}
				ctx.BindReg(r106, &d103)
			}
			ctx.EnsureDesc(&d103)
			ctx.EnsureDesc(&d103)
			var d104 scm.JITValueDesc
			if d103.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d103.Imm.Int()))))}
			} else {
				r107 := ctx.AllocReg()
				ctx.EmitMovRegReg(r107, d103.Reg)
				ctx.EmitShlRegImm8(r107, 56)
				ctx.EmitShrRegImm8(r107, 56)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d104)
			}
			ctx.FreeDesc(&d103)
			d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d104)
			var d106 scm.JITValueDesc
			if d105.Loc == scm.LocImm && d104.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d105.Imm.Int() - d104.Imm.Int())}
			} else if d104.Loc == scm.LocImm && d104.Imm.Int() == 0 {
				r108 := ctx.AllocRegExcept(d105.Reg)
				ctx.EmitMovRegReg(r108, d105.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d106)
			} else if d105.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d104.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d105.Imm.Int()))
				ctx.EmitSubInt64(scratch, d104.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d106)
			} else if d104.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d105.Reg)
				ctx.EmitMovRegReg(scratch, d105.Reg)
				if d104.Imm.Int() >= -2147483648 && d104.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d104.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d104.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d106)
			} else {
				r109 := ctx.AllocRegExcept(d105.Reg, d104.Reg)
				ctx.EmitMovRegReg(r109, d105.Reg)
				ctx.EmitSubInt64(r109, d104.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d106)
			}
			if d106.Loc == scm.LocReg && d105.Loc == scm.LocReg && d106.Reg == d105.Reg {
				ctx.TransferReg(d105.Reg)
				d105.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d104)
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d106)
			var d107 scm.JITValueDesc
			if d89.Loc == scm.LocImm && d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d89.Imm.Int()) >> uint64(d106.Imm.Int())))}
			} else if d106.Loc == scm.LocImm {
				r110 := ctx.AllocRegExcept(d89.Reg)
				ctx.EmitMovRegReg(r110, d89.Reg)
				ctx.EmitShrRegImm8(r110, uint8(d106.Imm.Int()))
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d107)
			} else {
				{
					shiftSrc := d89.Reg
					r111 := ctx.AllocRegExcept(d89.Reg)
					ctx.EmitMovRegReg(r111, d89.Reg)
					shiftSrc = r111
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d106.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d106.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d106.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d107)
				}
			}
			if d107.Loc == scm.LocReg && d89.Loc == scm.LocReg && d107.Reg == d89.Reg {
				ctx.TransferReg(d89.Reg)
				d89.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d89)
			ctx.FreeDesc(&d106)
			r112 := ctx.AllocReg()
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d107)
			if d107.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r112, d107)
			}
			ctx.EmitJmp(lbl34)
			bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl35)
			ctx.ResolveFixups()
			d89 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d93)
			var d108 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() % 64)}
			} else {
				r113 := ctx.AllocRegExcept(d93.Reg)
				ctx.EmitMovRegReg(r113, d93.Reg)
				ctx.EmitAndRegImm32(r113, 63)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d108)
			}
			if d108.Loc == scm.LocReg && d93.Loc == scm.LocReg && d108.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			var d109 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r114 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r114, thisptr.Reg, off)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d109)
			}
			ctx.EnsureDesc(&d109)
			ctx.EnsureDesc(&d109)
			var d110 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d109.Imm.Int()))))}
			} else {
				r115 := ctx.AllocReg()
				ctx.EmitMovRegReg(r115, d109.Reg)
				ctx.EmitShlRegImm8(r115, 56)
				ctx.EmitShrRegImm8(r115, 56)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d110)
			}
			ctx.FreeDesc(&d109)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d110)
			var d111 scm.JITValueDesc
			if d108.Loc == scm.LocImm && d110.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d108.Imm.Int() + d110.Imm.Int())}
			} else if d110.Loc == scm.LocImm && d110.Imm.Int() == 0 {
				r116 := ctx.AllocRegExcept(d108.Reg)
				ctx.EmitMovRegReg(r116, d108.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d111)
			} else if d108.Loc == scm.LocImm && d108.Imm.Int() == 0 {
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d110.Reg}
				ctx.BindReg(d110.Reg, &d111)
			} else if d108.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d110.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d108.Imm.Int()))
				ctx.EmitAddInt64(scratch, d110.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d111)
			} else if d110.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d108.Reg)
				ctx.EmitMovRegReg(scratch, d108.Reg)
				if d110.Imm.Int() >= -2147483648 && d110.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d110.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d111)
			} else {
				r117 := ctx.AllocRegExcept(d108.Reg, d110.Reg)
				ctx.EmitMovRegReg(r117, d108.Reg)
				ctx.EmitAddInt64(r117, d110.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d111)
			}
			if d111.Loc == scm.LocReg && d108.Loc == scm.LocReg && d111.Reg == d108.Reg {
				ctx.TransferReg(d108.Reg)
				d108.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d108)
			ctx.FreeDesc(&d110)
			ctx.EnsureDesc(&d111)
			var d112 scm.JITValueDesc
			if d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d111.Imm.Int()) > uint64(64))}
			} else {
				r118 := ctx.AllocRegExcept(d111.Reg)
				ctx.EmitCmpRegImm32(d111.Reg, 64)
				ctx.EmitSetcc(r118, scm.CcA)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r118}
				ctx.BindReg(r118, &d112)
			}
			ctx.FreeDesc(&d111)
			d113 = d112
			ctx.EnsureDesc(&d113)
			if d113.Loc != scm.LocImm && d113.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl39 := ctx.ReserveLabel()
			lbl40 := ctx.ReserveLabel()
			lbl41 := ctx.ReserveLabel()
			if d113.Loc == scm.LocImm {
				if d113.Imm.Bool() {
					ctx.MarkLabel(lbl40)
					ctx.EmitJmp(lbl39)
				} else {
					ctx.MarkLabel(lbl41)
			ctx.EnsureDesc(&d98)
			if d98.Loc == scm.LocReg {
				ctx.ProtectReg(d98.Reg)
			} else if d98.Loc == scm.LocRegPair {
				ctx.ProtectReg(d98.Reg)
				ctx.ProtectReg(d98.Reg2)
			}
			d114 = d98
			if d114.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d114)
			ctx.EmitStoreToStack(d114, int32(bbs[2].PhiBase)+int32(0))
			if d98.Loc == scm.LocReg {
				ctx.UnprotectReg(d98.Reg)
			} else if d98.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d98.Reg)
				ctx.UnprotectReg(d98.Reg2)
			}
					ctx.EmitJmp(lbl36)
				}
			} else {
				ctx.EmitCmpRegImm32(d113.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl40)
				ctx.EmitJmp(lbl41)
				ctx.MarkLabel(lbl40)
				ctx.EmitJmp(lbl39)
				ctx.MarkLabel(lbl41)
			ctx.EnsureDesc(&d98)
			if d98.Loc == scm.LocReg {
				ctx.ProtectReg(d98.Reg)
			} else if d98.Loc == scm.LocRegPair {
				ctx.ProtectReg(d98.Reg)
				ctx.ProtectReg(d98.Reg2)
			}
			d115 = d98
			if d115.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d115)
			ctx.EmitStoreToStack(d115, int32(bbs[2].PhiBase)+int32(0))
			if d98.Loc == scm.LocReg {
				ctx.UnprotectReg(d98.Reg)
			} else if d98.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d98.Reg)
				ctx.UnprotectReg(d98.Reg2)
			}
				ctx.EmitJmp(lbl36)
			}
			ctx.FreeDesc(&d112)
			bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl39)
			ctx.ResolveFixups()
			d89 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d93)
			var d116 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() / 64)}
			} else {
				r119 := ctx.AllocRegExcept(d93.Reg)
				ctx.EmitMovRegReg(r119, d93.Reg)
				ctx.EmitShrRegImm8(r119, 6)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d116)
			}
			if d116.Loc == scm.LocReg && d93.Loc == scm.LocReg && d116.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d116)
			ctx.EnsureDesc(&d116)
			var d117 scm.JITValueDesc
			if d116.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d116.Reg)
				ctx.EmitMovRegReg(scratch, d116.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d117)
			}
			if d117.Loc == scm.LocReg && d116.Loc == scm.LocReg && d117.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			ctx.EnsureDesc(&d117)
			r120 := ctx.AllocReg()
			ctx.EnsureDesc(&d117)
			ctx.EnsureDesc(&d94)
			if d117.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r120, uint64(d117.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r120, d117.Reg)
				ctx.EmitShlRegImm8(r120, 3)
			}
			if d94.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d94.Imm.Int()))
				ctx.EmitAddInt64(r120, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r120, d94.Reg)
			}
			r121 := ctx.AllocRegExcept(r120)
			ctx.EmitMovRegMem(r121, r120, 0)
			ctx.FreeReg(r120)
			d118 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			ctx.BindReg(r121, &d118)
			ctx.FreeDesc(&d117)
			ctx.EnsureDesc(&d93)
			var d119 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() % 64)}
			} else {
				r122 := ctx.AllocRegExcept(d93.Reg)
				ctx.EmitMovRegReg(r122, d93.Reg)
				ctx.EmitAndRegImm32(r122, 63)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d119)
			}
			if d119.Loc == scm.LocReg && d93.Loc == scm.LocReg && d119.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d93)
			d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d119)
			var d121 scm.JITValueDesc
			if d120.Loc == scm.LocImm && d119.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d120.Imm.Int() - d119.Imm.Int())}
			} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
				r123 := ctx.AllocRegExcept(d120.Reg)
				ctx.EmitMovRegReg(r123, d120.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d121)
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d119.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d120.Imm.Int()))
				ctx.EmitSubInt64(scratch, d119.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d121)
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d120.Reg)
				ctx.EmitMovRegReg(scratch, d120.Reg)
				if d119.Imm.Int() >= -2147483648 && d119.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d119.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d119.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d121)
			} else {
				r124 := ctx.AllocRegExcept(d120.Reg, d119.Reg)
				ctx.EmitMovRegReg(r124, d120.Reg)
				ctx.EmitSubInt64(r124, d119.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d121)
			}
			if d121.Loc == scm.LocReg && d120.Loc == scm.LocReg && d121.Reg == d120.Reg {
				ctx.TransferReg(d120.Reg)
				d120.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d121)
			var d122 scm.JITValueDesc
			if d118.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d118.Imm.Int()) >> uint64(d121.Imm.Int())))}
			} else if d121.Loc == scm.LocImm {
				r125 := ctx.AllocRegExcept(d118.Reg)
				ctx.EmitMovRegReg(r125, d118.Reg)
				ctx.EmitShrRegImm8(r125, uint8(d121.Imm.Int()))
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d122)
			} else {
				{
					shiftSrc := d118.Reg
					r126 := ctx.AllocRegExcept(d118.Reg)
					ctx.EmitMovRegReg(r126, d118.Reg)
					shiftSrc = r126
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d121.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d121.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d121.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d122)
				}
			}
			if d122.Loc == scm.LocReg && d118.Loc == scm.LocReg && d122.Reg == d118.Reg {
				ctx.TransferReg(d118.Reg)
				d118.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d118)
			ctx.FreeDesc(&d121)
			ctx.EnsureDesc(&d98)
			ctx.EnsureDesc(&d122)
			var d123 scm.JITValueDesc
			if d98.Loc == scm.LocImm && d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d98.Imm.Int() | d122.Imm.Int())}
			} else if d98.Loc == scm.LocImm && d98.Imm.Int() == 0 {
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d122.Reg}
				ctx.BindReg(d122.Reg, &d123)
			} else if d122.Loc == scm.LocImm && d122.Imm.Int() == 0 {
				r127 := ctx.AllocRegExcept(d98.Reg)
				ctx.EmitMovRegReg(r127, d98.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d123)
			} else if d98.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d122.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d98.Imm.Int()))
				ctx.EmitOrInt64(scratch, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d123)
			} else if d122.Loc == scm.LocImm {
				r128 := ctx.AllocRegExcept(d98.Reg)
				ctx.EmitMovRegReg(r128, d98.Reg)
				if d122.Imm.Int() >= -2147483648 && d122.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r128, int32(d122.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d122.Imm.Int()))
					ctx.EmitOrInt64(r128, scm.RegR11)
				}
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d123)
			} else {
				r129 := ctx.AllocRegExcept(d98.Reg, d122.Reg)
				ctx.EmitMovRegReg(r129, d98.Reg)
				ctx.EmitOrInt64(r129, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d123)
			}
			if d123.Loc == scm.LocReg && d98.Loc == scm.LocReg && d123.Reg == d98.Reg {
				ctx.TransferReg(d98.Reg)
				d98.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d122)
			ctx.EnsureDesc(&d123)
			if d123.Loc == scm.LocReg {
				ctx.ProtectReg(d123.Reg)
			} else if d123.Loc == scm.LocRegPair {
				ctx.ProtectReg(d123.Reg)
				ctx.ProtectReg(d123.Reg2)
			}
			d124 = d123
			if d124.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d124)
			ctx.EmitStoreToStack(d124, int32(bbs[2].PhiBase)+int32(0))
			if d123.Loc == scm.LocReg {
				ctx.UnprotectReg(d123.Reg)
			} else if d123.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d123.Reg)
				ctx.UnprotectReg(d123.Reg2)
			}
			ctx.EmitJmp(lbl36)
			ctx.MarkLabel(lbl34)
			d125 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
			ctx.BindReg(r112, &d125)
			ctx.BindReg(r112, &d125)
			if r92 { ctx.UnprotectReg(r93) }
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d125)
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d125.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.EmitMovRegReg(r130, d125.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d126)
			}
			ctx.FreeDesc(&d125)
			var d127 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r131 := ctx.AllocReg()
				ctx.EmitMovRegMem(r131, thisptr.Reg, off)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
				ctx.BindReg(r131, &d127)
			}
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d127)
			var d128 scm.JITValueDesc
			if d126.Loc == scm.LocImm && d127.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() + d127.Imm.Int())}
			} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d126.Reg)
				ctx.EmitMovRegReg(r132, d126.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d128)
			} else if d126.Loc == scm.LocImm && d126.Imm.Int() == 0 {
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d127.Reg}
				ctx.BindReg(d127.Reg, &d128)
			} else if d126.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d127.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d126.Imm.Int()))
				ctx.EmitAddInt64(scratch, d127.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d128)
			} else if d127.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d126.Reg)
				ctx.EmitMovRegReg(scratch, d126.Reg)
				if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d127.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d128)
			} else {
				r133 := ctx.AllocRegExcept(d126.Reg, d127.Reg)
				ctx.EmitMovRegReg(r133, d126.Reg)
				ctx.EmitAddInt64(r133, d127.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d128)
			}
			if d128.Loc == scm.LocReg && d126.Loc == scm.LocReg && d128.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d126)
			ctx.FreeDesc(&d127)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d128)
			var d130 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d128.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() + d128.Imm.Int())}
			} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
				r134 := ctx.AllocRegExcept(d87.Reg)
				ctx.EmitMovRegReg(r134, d87.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d130)
			} else if d87.Loc == scm.LocImm && d87.Imm.Int() == 0 {
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
				ctx.BindReg(d128.Reg, &d130)
			} else if d87.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d128.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d87.Imm.Int()))
				ctx.EmitAddInt64(scratch, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d87.Reg)
				ctx.EmitMovRegReg(scratch, d87.Reg)
				if d128.Imm.Int() >= -2147483648 && d128.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d128.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d128.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else {
				r135 := ctx.AllocRegExcept(d87.Reg, d128.Reg)
				ctx.EmitMovRegReg(r135, d87.Reg)
				ctx.EmitAddInt64(r135, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d130)
			}
			if d130.Loc == scm.LocReg && d87.Loc == scm.LocReg && d130.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d128)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d130)
			var d132 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 72
				r136 := ctx.AllocReg()
				r137 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r136, fieldAddr)
				ctx.EmitMovRegMem64(r137, fieldAddr+8)
				d132 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r136, Reg2: r137}
				ctx.BindReg(r136, &d132)
				ctx.BindReg(r137, &d132)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 72)
				r138 := ctx.AllocReg()
				r139 := ctx.AllocReg()
				ctx.EmitMovRegMem(r138, thisptr.Reg, off)
				ctx.EmitMovRegMem(r139, thisptr.Reg, off+8)
				d132 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r138, Reg2: r139}
				ctx.BindReg(r138, &d132)
				ctx.BindReg(r139, &d132)
			}
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d130)
			r140 := ctx.AllocReg()
			r141 := ctx.AllocRegExcept(r140)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d130)
			if d132.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r140, uint64(d132.Imm.Int()))
			} else if d132.Loc == scm.LocRegPair {
				ctx.EmitMovRegReg(r140, d132.Reg)
			} else {
				ctx.EmitMovRegReg(r140, d132.Reg)
			}
			if d87.Loc == scm.LocImm {
				if d87.Imm.Int() != 0 {
					if d87.Imm.Int() >= -2147483648 && d87.Imm.Int() <= 2147483647 {
						ctx.EmitAddRegImm32(r140, int32(d87.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
						ctx.EmitAddInt64(r140, scm.RegR11)
					}
				}
			} else {
				ctx.EmitAddInt64(r140, d87.Reg)
			}
			if d130.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r141, uint64(d130.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r141, d130.Reg)
			}
			if d87.Loc == scm.LocImm {
				if d87.Imm.Int() >= -2147483648 && d87.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(r141, int32(d87.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
					ctx.EmitSubInt64(r141, scm.RegR11)
				}
			} else {
				ctx.EmitSubInt64(r141, d87.Reg)
			}
			d133 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r140, Reg2: r141}
			ctx.BindReg(r140, &d133)
			ctx.BindReg(r141, &d133)
			ctx.FreeDesc(&d87)
			ctx.FreeDesc(&d130)
			r142 := ctx.AllocReg()
			r143 := ctx.AllocRegExcept(r142)
			d134 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d134)
			ctx.BindReg(r143, &d134)
			d135 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d133}, 2)
			ctx.EmitMovPairToResult(&d135, &d134)
			ctx.EmitJmp(lbl9)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl10)
			ctx.ResolveFixups()
			ctx.EnsureDesc(&d0)
			d136 = d0
			_ = d136
			r144 := d0.Loc == scm.LocReg
			r145 := d0.Reg
			if r144 { ctx.ProtectReg(r145) }
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			lbl42 := ctx.ReserveLabel()
			bbpos_5_0 := int32(-1)
			_ = bbpos_5_0
			bbpos_5_1 := int32(-1)
			_ = bbpos_5_1
			bbpos_5_2 := int32(-1)
			_ = bbpos_5_2
			bbpos_5_3 := int32(-1)
			_ = bbpos_5_3
			bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d136)
			var d138 scm.JITValueDesc
			if d136.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d136.Imm.Int()))))}
			} else {
				r146 := ctx.AllocReg()
				ctx.EmitMovRegReg(r146, d136.Reg)
				ctx.EmitShlRegImm8(r146, 32)
				ctx.EmitShrRegImm8(r146, 32)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d138)
			}
			var d139 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r147 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r147, thisptr.Reg, off)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r147}
				ctx.BindReg(r147, &d139)
			}
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d139)
			var d140 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d139.Imm.Int()))))}
			} else {
				r148 := ctx.AllocReg()
				ctx.EmitMovRegReg(r148, d139.Reg)
				ctx.EmitShlRegImm8(r148, 56)
				ctx.EmitShrRegImm8(r148, 56)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d140)
			}
			ctx.FreeDesc(&d139)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d140)
			var d141 scm.JITValueDesc
			if d138.Loc == scm.LocImm && d140.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d138.Imm.Int() * d140.Imm.Int())}
			} else if d138.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d140.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d138.Imm.Int()))
				ctx.EmitImulInt64(scratch, d140.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d141)
			} else if d140.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d138.Reg)
				ctx.EmitMovRegReg(scratch, d138.Reg)
				if d140.Imm.Int() >= -2147483648 && d140.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d140.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d140.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d141)
			} else {
				r149 := ctx.AllocRegExcept(d138.Reg, d140.Reg)
				ctx.EmitMovRegReg(r149, d138.Reg)
				ctx.EmitImulInt64(r149, d140.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d141)
			}
			if d141.Loc == scm.LocReg && d138.Loc == scm.LocReg && d141.Reg == d138.Reg {
				ctx.TransferReg(d138.Reg)
				d138.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d138)
			ctx.FreeDesc(&d140)
			var d142 scm.JITValueDesc
			r150 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r150, uint64(dataPtr))
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150, StackOff: int32(sliceLen)}
				ctx.BindReg(r150, &d142)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.EmitMovRegMem(r150, thisptr.Reg, off)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
				ctx.BindReg(r150, &d142)
			}
			ctx.BindReg(r150, &d142)
			ctx.EnsureDesc(&d141)
			var d143 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() / 64)}
			} else {
				r151 := ctx.AllocRegExcept(d141.Reg)
				ctx.EmitMovRegReg(r151, d141.Reg)
				ctx.EmitShrRegImm8(r151, 6)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d143)
			}
			if d143.Loc == scm.LocReg && d141.Loc == scm.LocReg && d143.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d143)
			r152 := ctx.AllocReg()
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d142)
			if d143.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r152, uint64(d143.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r152, d143.Reg)
				ctx.EmitShlRegImm8(r152, 3)
			}
			if d142.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d142.Imm.Int()))
				ctx.EmitAddInt64(r152, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r152, d142.Reg)
			}
			r153 := ctx.AllocRegExcept(r152)
			ctx.EmitMovRegMem(r153, r152, 0)
			ctx.FreeReg(r152)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r153}
			ctx.BindReg(r153, &d144)
			ctx.FreeDesc(&d143)
			ctx.EnsureDesc(&d141)
			var d145 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() % 64)}
			} else {
				r154 := ctx.AllocRegExcept(d141.Reg)
				ctx.EmitMovRegReg(r154, d141.Reg)
				ctx.EmitAndRegImm32(r154, 63)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d145)
			}
			if d145.Loc == scm.LocReg && d141.Loc == scm.LocReg && d145.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d145)
			var d146 scm.JITValueDesc
			if d144.Loc == scm.LocImm && d145.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d144.Imm.Int()) << uint64(d145.Imm.Int())))}
			} else if d145.Loc == scm.LocImm {
				r155 := ctx.AllocRegExcept(d144.Reg)
				ctx.EmitMovRegReg(r155, d144.Reg)
				ctx.EmitShlRegImm8(r155, uint8(d145.Imm.Int()))
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d146)
			} else {
				{
					shiftSrc := d144.Reg
					r156 := ctx.AllocRegExcept(d144.Reg)
					ctx.EmitMovRegReg(r156, d144.Reg)
					shiftSrc = r156
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d145.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d145.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d145.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d146)
				}
			}
			if d146.Loc == scm.LocReg && d144.Loc == scm.LocReg && d146.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d144)
			ctx.FreeDesc(&d145)
			var d147 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r157 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r157, thisptr.Reg, off)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r157}
				ctx.BindReg(r157, &d147)
			}
			d148 = d147
			ctx.EnsureDesc(&d148)
			if d148.Loc != scm.LocImm && d148.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl43 := ctx.ReserveLabel()
			lbl44 := ctx.ReserveLabel()
			lbl45 := ctx.ReserveLabel()
			lbl46 := ctx.ReserveLabel()
			if d148.Loc == scm.LocImm {
				if d148.Imm.Bool() {
					ctx.MarkLabel(lbl45)
					ctx.EmitJmp(lbl43)
				} else {
					ctx.MarkLabel(lbl46)
			ctx.EnsureDesc(&d146)
			if d146.Loc == scm.LocReg {
				ctx.ProtectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.ProtectReg(d146.Reg)
				ctx.ProtectReg(d146.Reg2)
			}
			d149 = d146
			if d149.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d149)
			ctx.EmitStoreToStack(d149, int32(bbs[2].PhiBase)+int32(0))
			if d146.Loc == scm.LocReg {
				ctx.UnprotectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d146.Reg)
				ctx.UnprotectReg(d146.Reg2)
			}
					ctx.EmitJmp(lbl44)
				}
			} else {
				ctx.EmitCmpRegImm32(d148.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl45)
				ctx.EmitJmp(lbl46)
				ctx.MarkLabel(lbl45)
				ctx.EmitJmp(lbl43)
				ctx.MarkLabel(lbl46)
			ctx.EnsureDesc(&d146)
			if d146.Loc == scm.LocReg {
				ctx.ProtectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.ProtectReg(d146.Reg)
				ctx.ProtectReg(d146.Reg2)
			}
			d150 = d146
			if d150.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d150)
			ctx.EmitStoreToStack(d150, int32(bbs[2].PhiBase)+int32(0))
			if d146.Loc == scm.LocReg {
				ctx.UnprotectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d146.Reg)
				ctx.UnprotectReg(d146.Reg2)
			}
				ctx.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d147)
			bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl44)
			ctx.ResolveFixups()
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			var d151 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r158 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r158, thisptr.Reg, off)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
				ctx.BindReg(r158, &d151)
			}
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d151)
			var d152 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d151.Imm.Int()))))}
			} else {
				r159 := ctx.AllocReg()
				ctx.EmitMovRegReg(r159, d151.Reg)
				ctx.EmitShlRegImm8(r159, 56)
				ctx.EmitShrRegImm8(r159, 56)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d152)
			}
			ctx.FreeDesc(&d151)
			d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d152)
			var d154 scm.JITValueDesc
			if d153.Loc == scm.LocImm && d152.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d153.Imm.Int() - d152.Imm.Int())}
			} else if d152.Loc == scm.LocImm && d152.Imm.Int() == 0 {
				r160 := ctx.AllocRegExcept(d153.Reg)
				ctx.EmitMovRegReg(r160, d153.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
				ctx.BindReg(r160, &d154)
			} else if d153.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d152.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d153.Imm.Int()))
				ctx.EmitSubInt64(scratch, d152.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d154)
			} else if d152.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d153.Reg)
				ctx.EmitMovRegReg(scratch, d153.Reg)
				if d152.Imm.Int() >= -2147483648 && d152.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d152.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d152.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d154)
			} else {
				r161 := ctx.AllocRegExcept(d153.Reg, d152.Reg)
				ctx.EmitMovRegReg(r161, d153.Reg)
				ctx.EmitSubInt64(r161, d152.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d154)
			}
			if d154.Loc == scm.LocReg && d153.Loc == scm.LocReg && d154.Reg == d153.Reg {
				ctx.TransferReg(d153.Reg)
				d153.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d152)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d154)
			var d155 scm.JITValueDesc
			if d137.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d137.Imm.Int()) >> uint64(d154.Imm.Int())))}
			} else if d154.Loc == scm.LocImm {
				r162 := ctx.AllocRegExcept(d137.Reg)
				ctx.EmitMovRegReg(r162, d137.Reg)
				ctx.EmitShrRegImm8(r162, uint8(d154.Imm.Int()))
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
				ctx.BindReg(r162, &d155)
			} else {
				{
					shiftSrc := d137.Reg
					r163 := ctx.AllocRegExcept(d137.Reg)
					ctx.EmitMovRegReg(r163, d137.Reg)
					shiftSrc = r163
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d154.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d154.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d154.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d155)
				}
			}
			if d155.Loc == scm.LocReg && d137.Loc == scm.LocReg && d155.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d137)
			ctx.FreeDesc(&d154)
			r164 := ctx.AllocReg()
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d155)
			if d155.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r164, d155)
			}
			ctx.EmitJmp(lbl42)
			bbpos_5_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl43)
			ctx.ResolveFixups()
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			ctx.EnsureDesc(&d141)
			var d156 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() % 64)}
			} else {
				r165 := ctx.AllocRegExcept(d141.Reg)
				ctx.EmitMovRegReg(r165, d141.Reg)
				ctx.EmitAndRegImm32(r165, 63)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d156)
			}
			if d156.Loc == scm.LocReg && d141.Loc == scm.LocReg && d156.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			var d157 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r166 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r166, thisptr.Reg, off)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r166}
				ctx.BindReg(r166, &d157)
			}
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d157)
			var d158 scm.JITValueDesc
			if d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d157.Imm.Int()))))}
			} else {
				r167 := ctx.AllocReg()
				ctx.EmitMovRegReg(r167, d157.Reg)
				ctx.EmitShlRegImm8(r167, 56)
				ctx.EmitShrRegImm8(r167, 56)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d158)
			}
			ctx.FreeDesc(&d157)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d158)
			var d159 scm.JITValueDesc
			if d156.Loc == scm.LocImm && d158.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() + d158.Imm.Int())}
			} else if d158.Loc == scm.LocImm && d158.Imm.Int() == 0 {
				r168 := ctx.AllocRegExcept(d156.Reg)
				ctx.EmitMovRegReg(r168, d156.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d159)
			} else if d156.Loc == scm.LocImm && d156.Imm.Int() == 0 {
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d158.Reg}
				ctx.BindReg(d158.Reg, &d159)
			} else if d156.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d158.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d156.Imm.Int()))
				ctx.EmitAddInt64(scratch, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			} else if d158.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d156.Reg)
				ctx.EmitMovRegReg(scratch, d156.Reg)
				if d158.Imm.Int() >= -2147483648 && d158.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d158.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d158.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			} else {
				r169 := ctx.AllocRegExcept(d156.Reg, d158.Reg)
				ctx.EmitMovRegReg(r169, d156.Reg)
				ctx.EmitAddInt64(r169, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d159)
			}
			if d159.Loc == scm.LocReg && d156.Loc == scm.LocReg && d159.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d156)
			ctx.FreeDesc(&d158)
			ctx.EnsureDesc(&d159)
			var d160 scm.JITValueDesc
			if d159.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d159.Imm.Int()) > uint64(64))}
			} else {
				r170 := ctx.AllocRegExcept(d159.Reg)
				ctx.EmitCmpRegImm32(d159.Reg, 64)
				ctx.EmitSetcc(r170, scm.CcA)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r170}
				ctx.BindReg(r170, &d160)
			}
			ctx.FreeDesc(&d159)
			d161 = d160
			ctx.EnsureDesc(&d161)
			if d161.Loc != scm.LocImm && d161.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl47 := ctx.ReserveLabel()
			lbl48 := ctx.ReserveLabel()
			lbl49 := ctx.ReserveLabel()
			if d161.Loc == scm.LocImm {
				if d161.Imm.Bool() {
					ctx.MarkLabel(lbl48)
					ctx.EmitJmp(lbl47)
				} else {
					ctx.MarkLabel(lbl49)
			ctx.EnsureDesc(&d146)
			if d146.Loc == scm.LocReg {
				ctx.ProtectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.ProtectReg(d146.Reg)
				ctx.ProtectReg(d146.Reg2)
			}
			d162 = d146
			if d162.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d162)
			ctx.EmitStoreToStack(d162, int32(bbs[2].PhiBase)+int32(0))
			if d146.Loc == scm.LocReg {
				ctx.UnprotectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d146.Reg)
				ctx.UnprotectReg(d146.Reg2)
			}
					ctx.EmitJmp(lbl44)
				}
			} else {
				ctx.EmitCmpRegImm32(d161.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl48)
				ctx.EmitJmp(lbl49)
				ctx.MarkLabel(lbl48)
				ctx.EmitJmp(lbl47)
				ctx.MarkLabel(lbl49)
			ctx.EnsureDesc(&d146)
			if d146.Loc == scm.LocReg {
				ctx.ProtectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.ProtectReg(d146.Reg)
				ctx.ProtectReg(d146.Reg2)
			}
			d163 = d146
			if d163.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d163)
			ctx.EmitStoreToStack(d163, int32(bbs[2].PhiBase)+int32(0))
			if d146.Loc == scm.LocReg {
				ctx.UnprotectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d146.Reg)
				ctx.UnprotectReg(d146.Reg2)
			}
				ctx.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d160)
			bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl47)
			ctx.ResolveFixups()
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			ctx.EnsureDesc(&d141)
			var d164 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() / 64)}
			} else {
				r171 := ctx.AllocRegExcept(d141.Reg)
				ctx.EmitMovRegReg(r171, d141.Reg)
				ctx.EmitShrRegImm8(r171, 6)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d164)
			}
			if d164.Loc == scm.LocReg && d141.Loc == scm.LocReg && d164.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d164)
			var d165 scm.JITValueDesc
			if d164.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d164.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d164.Reg)
				ctx.EmitMovRegReg(scratch, d164.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			}
			if d165.Loc == scm.LocReg && d164.Loc == scm.LocReg && d165.Reg == d164.Reg {
				ctx.TransferReg(d164.Reg)
				d164.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d164)
			ctx.EnsureDesc(&d165)
			r172 := ctx.AllocReg()
			ctx.EnsureDesc(&d165)
			ctx.EnsureDesc(&d142)
			if d165.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r172, uint64(d165.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r172, d165.Reg)
				ctx.EmitShlRegImm8(r172, 3)
			}
			if d142.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d142.Imm.Int()))
				ctx.EmitAddInt64(r172, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r172, d142.Reg)
			}
			r173 := ctx.AllocRegExcept(r172)
			ctx.EmitMovRegMem(r173, r172, 0)
			ctx.FreeReg(r172)
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r173}
			ctx.BindReg(r173, &d166)
			ctx.FreeDesc(&d165)
			ctx.EnsureDesc(&d141)
			var d167 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() % 64)}
			} else {
				r174 := ctx.AllocRegExcept(d141.Reg)
				ctx.EmitMovRegReg(r174, d141.Reg)
				ctx.EmitAndRegImm32(r174, 63)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d167)
			}
			if d167.Loc == scm.LocReg && d141.Loc == scm.LocReg && d167.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d141)
			d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d167)
			var d169 scm.JITValueDesc
			if d168.Loc == scm.LocImm && d167.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d168.Imm.Int() - d167.Imm.Int())}
			} else if d167.Loc == scm.LocImm && d167.Imm.Int() == 0 {
				r175 := ctx.AllocRegExcept(d168.Reg)
				ctx.EmitMovRegReg(r175, d168.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d169)
			} else if d168.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d167.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d168.Imm.Int()))
				ctx.EmitSubInt64(scratch, d167.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d169)
			} else if d167.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d168.Reg)
				ctx.EmitMovRegReg(scratch, d168.Reg)
				if d167.Imm.Int() >= -2147483648 && d167.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d167.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d167.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d169)
			} else {
				r176 := ctx.AllocRegExcept(d168.Reg, d167.Reg)
				ctx.EmitMovRegReg(r176, d168.Reg)
				ctx.EmitSubInt64(r176, d167.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d169)
			}
			if d169.Loc == scm.LocReg && d168.Loc == scm.LocReg && d169.Reg == d168.Reg {
				ctx.TransferReg(d168.Reg)
				d168.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			ctx.EnsureDesc(&d166)
			ctx.EnsureDesc(&d169)
			var d170 scm.JITValueDesc
			if d166.Loc == scm.LocImm && d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d166.Imm.Int()) >> uint64(d169.Imm.Int())))}
			} else if d169.Loc == scm.LocImm {
				r177 := ctx.AllocRegExcept(d166.Reg)
				ctx.EmitMovRegReg(r177, d166.Reg)
				ctx.EmitShrRegImm8(r177, uint8(d169.Imm.Int()))
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d170)
			} else {
				{
					shiftSrc := d166.Reg
					r178 := ctx.AllocRegExcept(d166.Reg)
					ctx.EmitMovRegReg(r178, d166.Reg)
					shiftSrc = r178
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d169.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d169.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d169.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d170)
				}
			}
			if d170.Loc == scm.LocReg && d166.Loc == scm.LocReg && d170.Reg == d166.Reg {
				ctx.TransferReg(d166.Reg)
				d166.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d166)
			ctx.FreeDesc(&d169)
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d170)
			var d171 scm.JITValueDesc
			if d146.Loc == scm.LocImm && d170.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d146.Imm.Int() | d170.Imm.Int())}
			} else if d146.Loc == scm.LocImm && d146.Imm.Int() == 0 {
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d170.Reg}
				ctx.BindReg(d170.Reg, &d171)
			} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
				r179 := ctx.AllocRegExcept(d146.Reg)
				ctx.EmitMovRegReg(r179, d146.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d171)
			} else if d146.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d170.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d146.Imm.Int()))
				ctx.EmitOrInt64(scratch, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d171)
			} else if d170.Loc == scm.LocImm {
				r180 := ctx.AllocRegExcept(d146.Reg)
				ctx.EmitMovRegReg(r180, d146.Reg)
				if d170.Imm.Int() >= -2147483648 && d170.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r180, int32(d170.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
					ctx.EmitOrInt64(r180, scm.RegR11)
				}
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
				ctx.BindReg(r180, &d171)
			} else {
				r181 := ctx.AllocRegExcept(d146.Reg, d170.Reg)
				ctx.EmitMovRegReg(r181, d146.Reg)
				ctx.EmitOrInt64(r181, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d171)
			}
			if d171.Loc == scm.LocReg && d146.Loc == scm.LocReg && d171.Reg == d146.Reg {
				ctx.TransferReg(d146.Reg)
				d146.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			ctx.EnsureDesc(&d171)
			if d171.Loc == scm.LocReg {
				ctx.ProtectReg(d171.Reg)
			} else if d171.Loc == scm.LocRegPair {
				ctx.ProtectReg(d171.Reg)
				ctx.ProtectReg(d171.Reg2)
			}
			d172 = d171
			if d172.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d172)
			ctx.EmitStoreToStack(d172, int32(bbs[2].PhiBase)+int32(0))
			if d171.Loc == scm.LocReg {
				ctx.UnprotectReg(d171.Reg)
			} else if d171.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d171.Reg)
				ctx.UnprotectReg(d171.Reg2)
			}
			ctx.EmitJmp(lbl44)
			ctx.MarkLabel(lbl42)
			d173 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r164}
			ctx.BindReg(r164, &d173)
			ctx.BindReg(r164, &d173)
			if r144 { ctx.UnprotectReg(r145) }
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d173)
			var d174 scm.JITValueDesc
			if d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d173.Imm.Int()))))}
			} else {
				r182 := ctx.AllocReg()
				ctx.EmitMovRegReg(r182, d173.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
				ctx.BindReg(r182, &d174)
			}
			ctx.FreeDesc(&d173)
			var d175 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r183 := ctx.AllocReg()
				ctx.EmitMovRegMem(r183, thisptr.Reg, off)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r183}
				ctx.BindReg(r183, &d175)
			}
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d175)
			var d176 scm.JITValueDesc
			if d174.Loc == scm.LocImm && d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() + d175.Imm.Int())}
			} else if d175.Loc == scm.LocImm && d175.Imm.Int() == 0 {
				r184 := ctx.AllocRegExcept(d174.Reg)
				ctx.EmitMovRegReg(r184, d174.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d176)
			} else if d174.Loc == scm.LocImm && d174.Imm.Int() == 0 {
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d175.Reg}
				ctx.BindReg(d175.Reg, &d176)
			} else if d174.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d175.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d174.Imm.Int()))
				ctx.EmitAddInt64(scratch, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d176)
			} else if d175.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d174.Reg)
				ctx.EmitMovRegReg(scratch, d174.Reg)
				if d175.Imm.Int() >= -2147483648 && d175.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d175.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d175.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d176)
			} else {
				r185 := ctx.AllocRegExcept(d174.Reg, d175.Reg)
				ctx.EmitMovRegReg(r185, d174.Reg)
				ctx.EmitAddInt64(r185, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d176)
			}
			if d176.Loc == scm.LocReg && d174.Loc == scm.LocReg && d176.Reg == d174.Reg {
				ctx.TransferReg(d174.Reg)
				d174.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d174)
			ctx.FreeDesc(&d175)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d176)
			var d177 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d176.Imm.Int()))))}
			} else {
				r186 := ctx.AllocReg()
				ctx.EmitMovRegReg(r186, d176.Reg)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
				ctx.BindReg(r186, &d177)
			}
			ctx.FreeDesc(&d176)
			var d178 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56)
				r187 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r187, thisptr.Reg, off)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r187}
				ctx.BindReg(r187, &d178)
			}
			d179 = d178
			ctx.EnsureDesc(&d179)
			if d179.Loc != scm.LocImm && d179.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl50 := ctx.ReserveLabel()
			lbl51 := ctx.ReserveLabel()
			lbl52 := ctx.ReserveLabel()
			lbl53 := ctx.ReserveLabel()
			if d179.Loc == scm.LocImm {
				if d179.Imm.Bool() {
					ctx.MarkLabel(lbl52)
					ctx.EmitJmp(lbl50)
				} else {
					ctx.MarkLabel(lbl53)
					ctx.EmitJmp(lbl51)
				}
			} else {
				ctx.EmitCmpRegImm32(d179.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl52)
				ctx.EmitJmp(lbl53)
				ctx.MarkLabel(lbl52)
				ctx.EmitJmp(lbl50)
				ctx.MarkLabel(lbl53)
				ctx.EmitJmp(lbl51)
			}
			ctx.FreeDesc(&d178)
			bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl51)
			ctx.ResolveFixups()
			ctx.EnsureDesc(&d0)
			d180 = d0
			_ = d180
			r188 := d0.Loc == scm.LocReg
			r189 := d0.Reg
			if r188 { ctx.ProtectReg(r189) }
			d181 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			lbl54 := ctx.ReserveLabel()
			bbpos_6_0 := int32(-1)
			_ = bbpos_6_0
			bbpos_6_1 := int32(-1)
			_ = bbpos_6_1
			bbpos_6_2 := int32(-1)
			_ = bbpos_6_2
			bbpos_6_3 := int32(-1)
			_ = bbpos_6_3
			bbpos_6_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d181 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d180)
			var d182 scm.JITValueDesc
			if d180.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d180.Imm.Int()))))}
			} else {
				r190 := ctx.AllocReg()
				ctx.EmitMovRegReg(r190, d180.Reg)
				ctx.EmitShlRegImm8(r190, 32)
				ctx.EmitShrRegImm8(r190, 32)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d182)
			}
			var d183 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r191 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r191, thisptr.Reg, off)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r191}
				ctx.BindReg(r191, &d183)
			}
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d183)
			var d184 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d183.Imm.Int()))))}
			} else {
				r192 := ctx.AllocReg()
				ctx.EmitMovRegReg(r192, d183.Reg)
				ctx.EmitShlRegImm8(r192, 56)
				ctx.EmitShrRegImm8(r192, 56)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d184)
			}
			ctx.FreeDesc(&d183)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d184)
			var d185 scm.JITValueDesc
			if d182.Loc == scm.LocImm && d184.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d182.Imm.Int() * d184.Imm.Int())}
			} else if d182.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d184.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d182.Imm.Int()))
				ctx.EmitImulInt64(scratch, d184.Reg)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d185)
			} else if d184.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d182.Reg)
				ctx.EmitMovRegReg(scratch, d182.Reg)
				if d184.Imm.Int() >= -2147483648 && d184.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d184.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d184.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d185)
			} else {
				r193 := ctx.AllocRegExcept(d182.Reg, d184.Reg)
				ctx.EmitMovRegReg(r193, d182.Reg)
				ctx.EmitImulInt64(r193, d184.Reg)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d185)
			}
			if d185.Loc == scm.LocReg && d182.Loc == scm.LocReg && d185.Reg == d182.Reg {
				ctx.TransferReg(d182.Reg)
				d182.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d182)
			ctx.FreeDesc(&d184)
			var d186 scm.JITValueDesc
			r194 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r194, uint64(dataPtr))
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r194, StackOff: int32(sliceLen)}
				ctx.BindReg(r194, &d186)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.EmitMovRegMem(r194, thisptr.Reg, off)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r194}
				ctx.BindReg(r194, &d186)
			}
			ctx.BindReg(r194, &d186)
			ctx.EnsureDesc(&d185)
			var d187 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() / 64)}
			} else {
				r195 := ctx.AllocRegExcept(d185.Reg)
				ctx.EmitMovRegReg(r195, d185.Reg)
				ctx.EmitShrRegImm8(r195, 6)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d187)
			}
			if d187.Loc == scm.LocReg && d185.Loc == scm.LocReg && d187.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d187)
			r196 := ctx.AllocReg()
			ctx.EnsureDesc(&d187)
			ctx.EnsureDesc(&d186)
			if d187.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r196, uint64(d187.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r196, d187.Reg)
				ctx.EmitShlRegImm8(r196, 3)
			}
			if d186.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d186.Imm.Int()))
				ctx.EmitAddInt64(r196, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r196, d186.Reg)
			}
			r197 := ctx.AllocRegExcept(r196)
			ctx.EmitMovRegMem(r197, r196, 0)
			ctx.FreeReg(r196)
			d188 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r197}
			ctx.BindReg(r197, &d188)
			ctx.FreeDesc(&d187)
			ctx.EnsureDesc(&d185)
			var d189 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() % 64)}
			} else {
				r198 := ctx.AllocRegExcept(d185.Reg)
				ctx.EmitMovRegReg(r198, d185.Reg)
				ctx.EmitAndRegImm32(r198, 63)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d189)
			}
			if d189.Loc == scm.LocReg && d185.Loc == scm.LocReg && d189.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d188)
			ctx.EnsureDesc(&d189)
			var d190 scm.JITValueDesc
			if d188.Loc == scm.LocImm && d189.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d188.Imm.Int()) << uint64(d189.Imm.Int())))}
			} else if d189.Loc == scm.LocImm {
				r199 := ctx.AllocRegExcept(d188.Reg)
				ctx.EmitMovRegReg(r199, d188.Reg)
				ctx.EmitShlRegImm8(r199, uint8(d189.Imm.Int()))
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d190)
			} else {
				{
					shiftSrc := d188.Reg
					r200 := ctx.AllocRegExcept(d188.Reg)
					ctx.EmitMovRegReg(r200, d188.Reg)
					shiftSrc = r200
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d189.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d189.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d189.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d190)
				}
			}
			if d190.Loc == scm.LocReg && d188.Loc == scm.LocReg && d190.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d188)
			ctx.FreeDesc(&d189)
			var d191 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r201 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r201, thisptr.Reg, off)
				d191 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r201}
				ctx.BindReg(r201, &d191)
			}
			d192 = d191
			ctx.EnsureDesc(&d192)
			if d192.Loc != scm.LocImm && d192.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl55 := ctx.ReserveLabel()
			lbl56 := ctx.ReserveLabel()
			lbl57 := ctx.ReserveLabel()
			lbl58 := ctx.ReserveLabel()
			if d192.Loc == scm.LocImm {
				if d192.Imm.Bool() {
					ctx.MarkLabel(lbl57)
					ctx.EmitJmp(lbl55)
				} else {
					ctx.MarkLabel(lbl58)
			ctx.EnsureDesc(&d190)
			if d190.Loc == scm.LocReg {
				ctx.ProtectReg(d190.Reg)
			} else if d190.Loc == scm.LocRegPair {
				ctx.ProtectReg(d190.Reg)
				ctx.ProtectReg(d190.Reg2)
			}
			d193 = d190
			if d193.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d193)
			ctx.EmitStoreToStack(d193, int32(bbs[2].PhiBase)+int32(0))
			if d190.Loc == scm.LocReg {
				ctx.UnprotectReg(d190.Reg)
			} else if d190.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d190.Reg)
				ctx.UnprotectReg(d190.Reg2)
			}
					ctx.EmitJmp(lbl56)
				}
			} else {
				ctx.EmitCmpRegImm32(d192.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl57)
				ctx.EmitJmp(lbl58)
				ctx.MarkLabel(lbl57)
				ctx.EmitJmp(lbl55)
				ctx.MarkLabel(lbl58)
			ctx.EnsureDesc(&d190)
			if d190.Loc == scm.LocReg {
				ctx.ProtectReg(d190.Reg)
			} else if d190.Loc == scm.LocRegPair {
				ctx.ProtectReg(d190.Reg)
				ctx.ProtectReg(d190.Reg2)
			}
			d194 = d190
			if d194.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d194)
			ctx.EmitStoreToStack(d194, int32(bbs[2].PhiBase)+int32(0))
			if d190.Loc == scm.LocReg {
				ctx.UnprotectReg(d190.Reg)
			} else if d190.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d190.Reg)
				ctx.UnprotectReg(d190.Reg2)
			}
				ctx.EmitJmp(lbl56)
			}
			ctx.FreeDesc(&d191)
			bbpos_6_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl56)
			ctx.ResolveFixups()
			d181 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			var d195 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r202 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r202, thisptr.Reg, off)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
				ctx.BindReg(r202, &d195)
			}
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d195)
			var d196 scm.JITValueDesc
			if d195.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d195.Imm.Int()))))}
			} else {
				r203 := ctx.AllocReg()
				ctx.EmitMovRegReg(r203, d195.Reg)
				ctx.EmitShlRegImm8(r203, 56)
				ctx.EmitShrRegImm8(r203, 56)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d196)
			}
			ctx.FreeDesc(&d195)
			d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d196)
			var d198 scm.JITValueDesc
			if d197.Loc == scm.LocImm && d196.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d197.Imm.Int() - d196.Imm.Int())}
			} else if d196.Loc == scm.LocImm && d196.Imm.Int() == 0 {
				r204 := ctx.AllocRegExcept(d197.Reg)
				ctx.EmitMovRegReg(r204, d197.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
				ctx.BindReg(r204, &d198)
			} else if d197.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d196.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d197.Imm.Int()))
				ctx.EmitSubInt64(scratch, d196.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d198)
			} else if d196.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d197.Reg)
				ctx.EmitMovRegReg(scratch, d197.Reg)
				if d196.Imm.Int() >= -2147483648 && d196.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d196.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d196.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d198)
			} else {
				r205 := ctx.AllocRegExcept(d197.Reg, d196.Reg)
				ctx.EmitMovRegReg(r205, d197.Reg)
				ctx.EmitSubInt64(r205, d196.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d198)
			}
			if d198.Loc == scm.LocReg && d197.Loc == scm.LocReg && d198.Reg == d197.Reg {
				ctx.TransferReg(d197.Reg)
				d197.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d196)
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d198)
			var d199 scm.JITValueDesc
			if d181.Loc == scm.LocImm && d198.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d181.Imm.Int()) >> uint64(d198.Imm.Int())))}
			} else if d198.Loc == scm.LocImm {
				r206 := ctx.AllocRegExcept(d181.Reg)
				ctx.EmitMovRegReg(r206, d181.Reg)
				ctx.EmitShrRegImm8(r206, uint8(d198.Imm.Int()))
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d199)
			} else {
				{
					shiftSrc := d181.Reg
					r207 := ctx.AllocRegExcept(d181.Reg)
					ctx.EmitMovRegReg(r207, d181.Reg)
					shiftSrc = r207
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d198.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d198.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d198.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d199)
				}
			}
			if d199.Loc == scm.LocReg && d181.Loc == scm.LocReg && d199.Reg == d181.Reg {
				ctx.TransferReg(d181.Reg)
				d181.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d181)
			ctx.FreeDesc(&d198)
			r208 := ctx.AllocReg()
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d199)
			if d199.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r208, d199)
			}
			ctx.EmitJmp(lbl54)
			bbpos_6_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl55)
			ctx.ResolveFixups()
			d181 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			ctx.EnsureDesc(&d185)
			var d200 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() % 64)}
			} else {
				r209 := ctx.AllocRegExcept(d185.Reg)
				ctx.EmitMovRegReg(r209, d185.Reg)
				ctx.EmitAndRegImm32(r209, 63)
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d200)
			}
			if d200.Loc == scm.LocReg && d185.Loc == scm.LocReg && d200.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			var d201 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r210 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r210, thisptr.Reg, off)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r210}
				ctx.BindReg(r210, &d201)
			}
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d201)
			var d202 scm.JITValueDesc
			if d201.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d201.Imm.Int()))))}
			} else {
				r211 := ctx.AllocReg()
				ctx.EmitMovRegReg(r211, d201.Reg)
				ctx.EmitShlRegImm8(r211, 56)
				ctx.EmitShrRegImm8(r211, 56)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d202)
			}
			ctx.FreeDesc(&d201)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d202)
			var d203 scm.JITValueDesc
			if d200.Loc == scm.LocImm && d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d200.Imm.Int() + d202.Imm.Int())}
			} else if d202.Loc == scm.LocImm && d202.Imm.Int() == 0 {
				r212 := ctx.AllocRegExcept(d200.Reg)
				ctx.EmitMovRegReg(r212, d200.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d203)
			} else if d200.Loc == scm.LocImm && d200.Imm.Int() == 0 {
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d202.Reg}
				ctx.BindReg(d202.Reg, &d203)
			} else if d200.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d202.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d200.Imm.Int()))
				ctx.EmitAddInt64(scratch, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else if d202.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d200.Reg)
				ctx.EmitMovRegReg(scratch, d200.Reg)
				if d202.Imm.Int() >= -2147483648 && d202.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d202.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d202.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else {
				r213 := ctx.AllocRegExcept(d200.Reg, d202.Reg)
				ctx.EmitMovRegReg(r213, d200.Reg)
				ctx.EmitAddInt64(r213, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d203)
			}
			if d203.Loc == scm.LocReg && d200.Loc == scm.LocReg && d203.Reg == d200.Reg {
				ctx.TransferReg(d200.Reg)
				d200.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d200)
			ctx.FreeDesc(&d202)
			ctx.EnsureDesc(&d203)
			var d204 scm.JITValueDesc
			if d203.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d203.Imm.Int()) > uint64(64))}
			} else {
				r214 := ctx.AllocRegExcept(d203.Reg)
				ctx.EmitCmpRegImm32(d203.Reg, 64)
				ctx.EmitSetcc(r214, scm.CcA)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r214}
				ctx.BindReg(r214, &d204)
			}
			ctx.FreeDesc(&d203)
			d205 = d204
			ctx.EnsureDesc(&d205)
			if d205.Loc != scm.LocImm && d205.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl59 := ctx.ReserveLabel()
			lbl60 := ctx.ReserveLabel()
			lbl61 := ctx.ReserveLabel()
			if d205.Loc == scm.LocImm {
				if d205.Imm.Bool() {
					ctx.MarkLabel(lbl60)
					ctx.EmitJmp(lbl59)
				} else {
					ctx.MarkLabel(lbl61)
			ctx.EnsureDesc(&d190)
			if d190.Loc == scm.LocReg {
				ctx.ProtectReg(d190.Reg)
			} else if d190.Loc == scm.LocRegPair {
				ctx.ProtectReg(d190.Reg)
				ctx.ProtectReg(d190.Reg2)
			}
			d206 = d190
			if d206.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d206)
			ctx.EmitStoreToStack(d206, int32(bbs[2].PhiBase)+int32(0))
			if d190.Loc == scm.LocReg {
				ctx.UnprotectReg(d190.Reg)
			} else if d190.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d190.Reg)
				ctx.UnprotectReg(d190.Reg2)
			}
					ctx.EmitJmp(lbl56)
				}
			} else {
				ctx.EmitCmpRegImm32(d205.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl60)
				ctx.EmitJmp(lbl61)
				ctx.MarkLabel(lbl60)
				ctx.EmitJmp(lbl59)
				ctx.MarkLabel(lbl61)
			ctx.EnsureDesc(&d190)
			if d190.Loc == scm.LocReg {
				ctx.ProtectReg(d190.Reg)
			} else if d190.Loc == scm.LocRegPair {
				ctx.ProtectReg(d190.Reg)
				ctx.ProtectReg(d190.Reg2)
			}
			d207 = d190
			if d207.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d207)
			ctx.EmitStoreToStack(d207, int32(bbs[2].PhiBase)+int32(0))
			if d190.Loc == scm.LocReg {
				ctx.UnprotectReg(d190.Reg)
			} else if d190.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d190.Reg)
				ctx.UnprotectReg(d190.Reg2)
			}
				ctx.EmitJmp(lbl56)
			}
			ctx.FreeDesc(&d204)
			bbpos_6_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl59)
			ctx.ResolveFixups()
			d181 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			ctx.EnsureDesc(&d185)
			var d208 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() / 64)}
			} else {
				r215 := ctx.AllocRegExcept(d185.Reg)
				ctx.EmitMovRegReg(r215, d185.Reg)
				ctx.EmitShrRegImm8(r215, 6)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d208)
			}
			if d208.Loc == scm.LocReg && d185.Loc == scm.LocReg && d208.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d208)
			ctx.EnsureDesc(&d208)
			var d209 scm.JITValueDesc
			if d208.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d208.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d208.Reg)
				ctx.EmitMovRegReg(scratch, d208.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			}
			if d209.Loc == scm.LocReg && d208.Loc == scm.LocReg && d209.Reg == d208.Reg {
				ctx.TransferReg(d208.Reg)
				d208.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d208)
			ctx.EnsureDesc(&d209)
			r216 := ctx.AllocReg()
			ctx.EnsureDesc(&d209)
			ctx.EnsureDesc(&d186)
			if d209.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r216, uint64(d209.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r216, d209.Reg)
				ctx.EmitShlRegImm8(r216, 3)
			}
			if d186.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d186.Imm.Int()))
				ctx.EmitAddInt64(r216, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r216, d186.Reg)
			}
			r217 := ctx.AllocRegExcept(r216)
			ctx.EmitMovRegMem(r217, r216, 0)
			ctx.FreeReg(r216)
			d210 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r217}
			ctx.BindReg(r217, &d210)
			ctx.FreeDesc(&d209)
			ctx.EnsureDesc(&d185)
			var d211 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() % 64)}
			} else {
				r218 := ctx.AllocRegExcept(d185.Reg)
				ctx.EmitMovRegReg(r218, d185.Reg)
				ctx.EmitAndRegImm32(r218, 63)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d211)
			}
			if d211.Loc == scm.LocReg && d185.Loc == scm.LocReg && d211.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d185)
			d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d211)
			var d213 scm.JITValueDesc
			if d212.Loc == scm.LocImm && d211.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() - d211.Imm.Int())}
			} else if d211.Loc == scm.LocImm && d211.Imm.Int() == 0 {
				r219 := ctx.AllocRegExcept(d212.Reg)
				ctx.EmitMovRegReg(r219, d212.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d213)
			} else if d212.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d211.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d212.Imm.Int()))
				ctx.EmitSubInt64(scratch, d211.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else if d211.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d212.Reg)
				ctx.EmitMovRegReg(scratch, d212.Reg)
				if d211.Imm.Int() >= -2147483648 && d211.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d211.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d211.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else {
				r220 := ctx.AllocRegExcept(d212.Reg, d211.Reg)
				ctx.EmitMovRegReg(r220, d212.Reg)
				ctx.EmitSubInt64(r220, d211.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d213)
			}
			if d213.Loc == scm.LocReg && d212.Loc == scm.LocReg && d213.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d211)
			ctx.EnsureDesc(&d210)
			ctx.EnsureDesc(&d213)
			var d214 scm.JITValueDesc
			if d210.Loc == scm.LocImm && d213.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d210.Imm.Int()) >> uint64(d213.Imm.Int())))}
			} else if d213.Loc == scm.LocImm {
				r221 := ctx.AllocRegExcept(d210.Reg)
				ctx.EmitMovRegReg(r221, d210.Reg)
				ctx.EmitShrRegImm8(r221, uint8(d213.Imm.Int()))
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d214)
			} else {
				{
					shiftSrc := d210.Reg
					r222 := ctx.AllocRegExcept(d210.Reg)
					ctx.EmitMovRegReg(r222, d210.Reg)
					shiftSrc = r222
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d213.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d213.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d213.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d214)
				}
			}
			if d214.Loc == scm.LocReg && d210.Loc == scm.LocReg && d214.Reg == d210.Reg {
				ctx.TransferReg(d210.Reg)
				d210.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d210)
			ctx.FreeDesc(&d213)
			ctx.EnsureDesc(&d190)
			ctx.EnsureDesc(&d214)
			var d215 scm.JITValueDesc
			if d190.Loc == scm.LocImm && d214.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d190.Imm.Int() | d214.Imm.Int())}
			} else if d190.Loc == scm.LocImm && d190.Imm.Int() == 0 {
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d214.Reg}
				ctx.BindReg(d214.Reg, &d215)
			} else if d214.Loc == scm.LocImm && d214.Imm.Int() == 0 {
				r223 := ctx.AllocRegExcept(d190.Reg)
				ctx.EmitMovRegReg(r223, d190.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d215)
			} else if d190.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d214.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d190.Imm.Int()))
				ctx.EmitOrInt64(scratch, d214.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d215)
			} else if d214.Loc == scm.LocImm {
				r224 := ctx.AllocRegExcept(d190.Reg)
				ctx.EmitMovRegReg(r224, d190.Reg)
				if d214.Imm.Int() >= -2147483648 && d214.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r224, int32(d214.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d214.Imm.Int()))
					ctx.EmitOrInt64(r224, scm.RegR11)
				}
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d215)
			} else {
				r225 := ctx.AllocRegExcept(d190.Reg, d214.Reg)
				ctx.EmitMovRegReg(r225, d190.Reg)
				ctx.EmitOrInt64(r225, d214.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d215)
			}
			if d215.Loc == scm.LocReg && d190.Loc == scm.LocReg && d215.Reg == d190.Reg {
				ctx.TransferReg(d190.Reg)
				d190.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d214)
			ctx.EnsureDesc(&d215)
			if d215.Loc == scm.LocReg {
				ctx.ProtectReg(d215.Reg)
			} else if d215.Loc == scm.LocRegPair {
				ctx.ProtectReg(d215.Reg)
				ctx.ProtectReg(d215.Reg2)
			}
			d216 = d215
			if d216.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d216)
			ctx.EmitStoreToStack(d216, int32(bbs[2].PhiBase)+int32(0))
			if d215.Loc == scm.LocReg {
				ctx.UnprotectReg(d215.Reg)
			} else if d215.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d215.Reg)
				ctx.UnprotectReg(d215.Reg2)
			}
			ctx.EmitJmp(lbl56)
			ctx.MarkLabel(lbl54)
			d217 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r208}
			ctx.BindReg(r208, &d217)
			ctx.BindReg(r208, &d217)
			if r188 { ctx.UnprotectReg(r189) }
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d217)
			var d218 scm.JITValueDesc
			if d217.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d217.Imm.Int()))))}
			} else {
				r226 := ctx.AllocReg()
				ctx.EmitMovRegReg(r226, d217.Reg)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d218)
			}
			ctx.FreeDesc(&d217)
			var d219 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r227 := ctx.AllocReg()
				ctx.EmitMovRegMem(r227, thisptr.Reg, off)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r227}
				ctx.BindReg(r227, &d219)
			}
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			var d220 scm.JITValueDesc
			if d218.Loc == scm.LocImm && d219.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d218.Imm.Int() + d219.Imm.Int())}
			} else if d219.Loc == scm.LocImm && d219.Imm.Int() == 0 {
				r228 := ctx.AllocRegExcept(d218.Reg)
				ctx.EmitMovRegReg(r228, d218.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d220)
			} else if d218.Loc == scm.LocImm && d218.Imm.Int() == 0 {
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d219.Reg}
				ctx.BindReg(d219.Reg, &d220)
			} else if d218.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d219.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d218.Imm.Int()))
				ctx.EmitAddInt64(scratch, d219.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d220)
			} else if d219.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d218.Reg)
				ctx.EmitMovRegReg(scratch, d218.Reg)
				if d219.Imm.Int() >= -2147483648 && d219.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d219.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d219.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d220)
			} else {
				r229 := ctx.AllocRegExcept(d218.Reg, d219.Reg)
				ctx.EmitMovRegReg(r229, d218.Reg)
				ctx.EmitAddInt64(r229, d219.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d220)
			}
			if d220.Loc == scm.LocReg && d218.Loc == scm.LocReg && d220.Reg == d218.Reg {
				ctx.TransferReg(d218.Reg)
				d218.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d218)
			ctx.FreeDesc(&d219)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d220)
			var d221 scm.JITValueDesc
			if d220.Loc == scm.LocImm {
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d220.Imm.Int()))))}
			} else {
				r230 := ctx.AllocReg()
				ctx.EmitMovRegReg(r230, d220.Reg)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d221)
			}
			ctx.FreeDesc(&d220)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d177)
			var d222 scm.JITValueDesc
			if d177.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d177.Imm.Int()))))}
			} else {
				r231 := ctx.AllocReg()
				ctx.EmitMovRegReg(r231, d177.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d222)
			}
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d221)
			var d223 scm.JITValueDesc
			if d177.Loc == scm.LocImm && d221.Loc == scm.LocImm {
				d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d177.Imm.Int() + d221.Imm.Int())}
			} else if d221.Loc == scm.LocImm && d221.Imm.Int() == 0 {
				r232 := ctx.AllocRegExcept(d177.Reg)
				ctx.EmitMovRegReg(r232, d177.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d223)
			} else if d177.Loc == scm.LocImm && d177.Imm.Int() == 0 {
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d221.Reg}
				ctx.BindReg(d221.Reg, &d223)
			} else if d177.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d221.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d177.Imm.Int()))
				ctx.EmitAddInt64(scratch, d221.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d223)
			} else if d221.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d177.Reg)
				ctx.EmitMovRegReg(scratch, d177.Reg)
				if d221.Imm.Int() >= -2147483648 && d221.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d221.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d221.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d223)
			} else {
				r233 := ctx.AllocRegExcept(d177.Reg, d221.Reg)
				ctx.EmitMovRegReg(r233, d177.Reg)
				ctx.EmitAddInt64(r233, d221.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d223)
			}
			if d223.Loc == scm.LocReg && d177.Loc == scm.LocReg && d223.Reg == d177.Reg {
				ctx.TransferReg(d177.Reg)
				d177.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d221)
			ctx.EnsureDesc(&d223)
			ctx.EnsureDesc(&d223)
			var d224 scm.JITValueDesc
			if d223.Loc == scm.LocImm {
				d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d223.Imm.Int()))))}
			} else {
				r234 := ctx.AllocReg()
				ctx.EmitMovRegReg(r234, d223.Reg)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d224)
			}
			ctx.FreeDesc(&d223)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			r235 := ctx.AllocReg()
			r236 := ctx.AllocRegExcept(r235)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			if d132.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r235, uint64(d132.Imm.Int()))
			} else if d132.Loc == scm.LocRegPair {
				ctx.EmitMovRegReg(r235, d132.Reg)
			} else {
				ctx.EmitMovRegReg(r235, d132.Reg)
			}
			if d222.Loc == scm.LocImm {
				if d222.Imm.Int() != 0 {
					if d222.Imm.Int() >= -2147483648 && d222.Imm.Int() <= 2147483647 {
						ctx.EmitAddRegImm32(r235, int32(d222.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
						ctx.EmitAddInt64(r235, scm.RegR11)
					}
				}
			} else {
				ctx.EmitAddInt64(r235, d222.Reg)
			}
			if d224.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r236, uint64(d224.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r236, d224.Reg)
			}
			if d222.Loc == scm.LocImm {
				if d222.Imm.Int() >= -2147483648 && d222.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(r236, int32(d222.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
					ctx.EmitSubInt64(r236, scm.RegR11)
				}
			} else {
				ctx.EmitSubInt64(r236, d222.Reg)
			}
			d225 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r235, Reg2: r236}
			ctx.BindReg(r235, &d225)
			ctx.BindReg(r236, &d225)
			ctx.FreeDesc(&d222)
			ctx.FreeDesc(&d224)
			d226 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d226)
			ctx.BindReg(r143, &d226)
			d227 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d225}, 2)
			ctx.EmitMovPairToResult(&d227, &d226)
			ctx.EmitJmp(lbl9)
			bbpos_1_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl22)
			ctx.ResolveFixups()
			var d228 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64)
				r237 := ctx.AllocReg()
				ctx.EmitMovRegMem(r237, thisptr.Reg, off)
				d228 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r237}
				ctx.BindReg(r237, &d228)
			}
			ctx.EnsureDesc(&d228)
			ctx.EnsureDesc(&d228)
			var d229 scm.JITValueDesc
			if d228.Loc == scm.LocImm {
				d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d228.Imm.Int()))))}
			} else {
				r238 := ctx.AllocReg()
				ctx.EmitMovRegReg(r238, d228.Reg)
				ctx.EmitShlRegImm8(r238, 32)
				ctx.EmitShrRegImm8(r238, 32)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d229)
			}
			ctx.FreeDesc(&d228)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d229)
			var d230 scm.JITValueDesc
			if d44.Loc == scm.LocImm && d229.Loc == scm.LocImm {
				d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d44.Imm.Int()) == uint64(d229.Imm.Int()))}
			} else if d229.Loc == scm.LocImm {
				r239 := ctx.AllocRegExcept(d44.Reg)
				if d229.Imm.Int() >= -2147483648 && d229.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d44.Reg, int32(d229.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d229.Imm.Int()))
					ctx.EmitCmpInt64(d44.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r239, scm.CcE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r239}
				ctx.BindReg(r239, &d230)
			} else if d44.Loc == scm.LocImm {
				r240 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d44.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d229.Reg)
				ctx.EmitSetcc(r240, scm.CcE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r240}
				ctx.BindReg(r240, &d230)
			} else {
				r241 := ctx.AllocRegExcept(d44.Reg)
				ctx.EmitCmpInt64(d44.Reg, d229.Reg)
				ctx.EmitSetcc(r241, scm.CcE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d230)
			}
			ctx.FreeDesc(&d44)
			ctx.FreeDesc(&d229)
			d231 = d230
			ctx.EnsureDesc(&d231)
			if d231.Loc != scm.LocImm && d231.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl62 := ctx.ReserveLabel()
			lbl63 := ctx.ReserveLabel()
			lbl64 := ctx.ReserveLabel()
			if d231.Loc == scm.LocImm {
				if d231.Imm.Bool() {
					ctx.MarkLabel(lbl63)
					ctx.EmitJmp(lbl62)
				} else {
					ctx.MarkLabel(lbl64)
					ctx.EmitJmp(lbl23)
				}
			} else {
				ctx.EmitCmpRegImm32(d231.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl63)
				ctx.EmitJmp(lbl64)
				ctx.MarkLabel(lbl63)
				ctx.EmitJmp(lbl62)
				ctx.MarkLabel(lbl64)
				ctx.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d230)
			bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl50)
			ctx.ResolveFixups()
			var d232 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64)
				r242 := ctx.AllocReg()
				ctx.EmitMovRegMem(r242, thisptr.Reg, off)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r242}
				ctx.BindReg(r242, &d232)
			}
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d232)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d232)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d232)
			var d233 scm.JITValueDesc
			if d177.Loc == scm.LocImm && d232.Loc == scm.LocImm {
				d233 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d177.Imm.Int()) == uint64(d232.Imm.Int()))}
			} else if d232.Loc == scm.LocImm {
				r243 := ctx.AllocRegExcept(d177.Reg)
				if d232.Imm.Int() >= -2147483648 && d232.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d177.Reg, int32(d232.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d232.Imm.Int()))
					ctx.EmitCmpInt64(d177.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r243, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r243}
				ctx.BindReg(r243, &d233)
			} else if d177.Loc == scm.LocImm {
				r244 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d232.Reg)
				ctx.EmitSetcc(r244, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r244}
				ctx.BindReg(r244, &d233)
			} else {
				r245 := ctx.AllocRegExcept(d177.Reg)
				ctx.EmitCmpInt64(d177.Reg, d232.Reg)
				ctx.EmitSetcc(r245, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r245}
				ctx.BindReg(r245, &d233)
			}
			ctx.FreeDesc(&d177)
			ctx.FreeDesc(&d232)
			d234 = d233
			ctx.EnsureDesc(&d234)
			if d234.Loc != scm.LocImm && d234.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl65 := ctx.ReserveLabel()
			lbl66 := ctx.ReserveLabel()
			lbl67 := ctx.ReserveLabel()
			if d234.Loc == scm.LocImm {
				if d234.Imm.Bool() {
					ctx.MarkLabel(lbl66)
					ctx.EmitJmp(lbl65)
				} else {
					ctx.MarkLabel(lbl67)
					ctx.EmitJmp(lbl51)
				}
			} else {
				ctx.EmitCmpRegImm32(d234.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl66)
				ctx.EmitJmp(lbl67)
				ctx.MarkLabel(lbl66)
				ctx.EmitJmp(lbl65)
				ctx.MarkLabel(lbl67)
				ctx.EmitJmp(lbl51)
			}
			ctx.FreeDesc(&d233)
			bbpos_1_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl62)
			ctx.ResolveFixups()
			d235 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d235)
			ctx.BindReg(r143, &d235)
			ctx.EmitMakeNil(d235)
			ctx.EmitJmp(lbl9)
			bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl65)
			ctx.ResolveFixups()
			d236 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d236)
			ctx.BindReg(r143, &d236)
			ctx.EmitMakeNil(d236)
			ctx.EmitJmp(lbl9)
			ctx.MarkLabel(lbl9)
			d237 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d237)
			ctx.BindReg(r143, &d237)
			ctx.BindReg(r142, &d237)
			ctx.BindReg(r143, &d237)
			if r2 { ctx.UnprotectReg(r3) }
			d239 = d237
			d239.ID = 0
			d238 = ctx.EmitTagEqualsBorrowed(&d239, scm.TagNil, scm.JITValueDesc{Loc: scm.LocAny})
			d240 = d238
			ctx.EnsureDesc(&d240)
			if d240.Loc != scm.LocImm && d240.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d240.Loc == scm.LocImm {
				if d240.Imm.Bool() {
			ps241 := scm.PhiState{General: ps.General}
			ps241.OverlayValues = make([]scm.JITValueDesc, 241)
			ps241.OverlayValues[0] = d0
			ps241.OverlayValues[1] = d1
			ps241.OverlayValues[2] = d2
			ps241.OverlayValues[3] = d3
			ps241.OverlayValues[4] = d4
			ps241.OverlayValues[5] = d5
			ps241.OverlayValues[6] = d6
			ps241.OverlayValues[7] = d7
			ps241.OverlayValues[8] = d8
			ps241.OverlayValues[9] = d9
			ps241.OverlayValues[10] = d10
			ps241.OverlayValues[11] = d11
			ps241.OverlayValues[12] = d12
			ps241.OverlayValues[13] = d13
			ps241.OverlayValues[14] = d14
			ps241.OverlayValues[15] = d15
			ps241.OverlayValues[16] = d16
			ps241.OverlayValues[17] = d17
			ps241.OverlayValues[18] = d18
			ps241.OverlayValues[19] = d19
			ps241.OverlayValues[20] = d20
			ps241.OverlayValues[21] = d21
			ps241.OverlayValues[22] = d22
			ps241.OverlayValues[23] = d23
			ps241.OverlayValues[24] = d24
			ps241.OverlayValues[25] = d25
			ps241.OverlayValues[26] = d26
			ps241.OverlayValues[27] = d27
			ps241.OverlayValues[28] = d28
			ps241.OverlayValues[29] = d29
			ps241.OverlayValues[30] = d30
			ps241.OverlayValues[31] = d31
			ps241.OverlayValues[32] = d32
			ps241.OverlayValues[33] = d33
			ps241.OverlayValues[34] = d34
			ps241.OverlayValues[35] = d35
			ps241.OverlayValues[36] = d36
			ps241.OverlayValues[37] = d37
			ps241.OverlayValues[38] = d38
			ps241.OverlayValues[39] = d39
			ps241.OverlayValues[40] = d40
			ps241.OverlayValues[41] = d41
			ps241.OverlayValues[42] = d42
			ps241.OverlayValues[43] = d43
			ps241.OverlayValues[44] = d44
			ps241.OverlayValues[45] = d45
			ps241.OverlayValues[46] = d46
			ps241.OverlayValues[47] = d47
			ps241.OverlayValues[48] = d48
			ps241.OverlayValues[49] = d49
			ps241.OverlayValues[50] = d50
			ps241.OverlayValues[51] = d51
			ps241.OverlayValues[52] = d52
			ps241.OverlayValues[53] = d53
			ps241.OverlayValues[54] = d54
			ps241.OverlayValues[55] = d55
			ps241.OverlayValues[56] = d56
			ps241.OverlayValues[57] = d57
			ps241.OverlayValues[58] = d58
			ps241.OverlayValues[59] = d59
			ps241.OverlayValues[60] = d60
			ps241.OverlayValues[61] = d61
			ps241.OverlayValues[62] = d62
			ps241.OverlayValues[63] = d63
			ps241.OverlayValues[64] = d64
			ps241.OverlayValues[65] = d65
			ps241.OverlayValues[66] = d66
			ps241.OverlayValues[67] = d67
			ps241.OverlayValues[68] = d68
			ps241.OverlayValues[69] = d69
			ps241.OverlayValues[70] = d70
			ps241.OverlayValues[71] = d71
			ps241.OverlayValues[72] = d72
			ps241.OverlayValues[73] = d73
			ps241.OverlayValues[74] = d74
			ps241.OverlayValues[75] = d75
			ps241.OverlayValues[76] = d76
			ps241.OverlayValues[77] = d77
			ps241.OverlayValues[78] = d78
			ps241.OverlayValues[79] = d79
			ps241.OverlayValues[80] = d80
			ps241.OverlayValues[81] = d81
			ps241.OverlayValues[82] = d82
			ps241.OverlayValues[83] = d83
			ps241.OverlayValues[84] = d84
			ps241.OverlayValues[85] = d85
			ps241.OverlayValues[86] = d86
			ps241.OverlayValues[87] = d87
			ps241.OverlayValues[88] = d88
			ps241.OverlayValues[89] = d89
			ps241.OverlayValues[90] = d90
			ps241.OverlayValues[91] = d91
			ps241.OverlayValues[92] = d92
			ps241.OverlayValues[93] = d93
			ps241.OverlayValues[94] = d94
			ps241.OverlayValues[95] = d95
			ps241.OverlayValues[96] = d96
			ps241.OverlayValues[97] = d97
			ps241.OverlayValues[98] = d98
			ps241.OverlayValues[99] = d99
			ps241.OverlayValues[100] = d100
			ps241.OverlayValues[101] = d101
			ps241.OverlayValues[102] = d102
			ps241.OverlayValues[103] = d103
			ps241.OverlayValues[104] = d104
			ps241.OverlayValues[105] = d105
			ps241.OverlayValues[106] = d106
			ps241.OverlayValues[107] = d107
			ps241.OverlayValues[108] = d108
			ps241.OverlayValues[109] = d109
			ps241.OverlayValues[110] = d110
			ps241.OverlayValues[111] = d111
			ps241.OverlayValues[112] = d112
			ps241.OverlayValues[113] = d113
			ps241.OverlayValues[114] = d114
			ps241.OverlayValues[115] = d115
			ps241.OverlayValues[116] = d116
			ps241.OverlayValues[117] = d117
			ps241.OverlayValues[118] = d118
			ps241.OverlayValues[119] = d119
			ps241.OverlayValues[120] = d120
			ps241.OverlayValues[121] = d121
			ps241.OverlayValues[122] = d122
			ps241.OverlayValues[123] = d123
			ps241.OverlayValues[124] = d124
			ps241.OverlayValues[125] = d125
			ps241.OverlayValues[126] = d126
			ps241.OverlayValues[127] = d127
			ps241.OverlayValues[128] = d128
			ps241.OverlayValues[129] = d129
			ps241.OverlayValues[130] = d130
			ps241.OverlayValues[131] = d131
			ps241.OverlayValues[132] = d132
			ps241.OverlayValues[133] = d133
			ps241.OverlayValues[134] = d134
			ps241.OverlayValues[135] = d135
			ps241.OverlayValues[136] = d136
			ps241.OverlayValues[137] = d137
			ps241.OverlayValues[138] = d138
			ps241.OverlayValues[139] = d139
			ps241.OverlayValues[140] = d140
			ps241.OverlayValues[141] = d141
			ps241.OverlayValues[142] = d142
			ps241.OverlayValues[143] = d143
			ps241.OverlayValues[144] = d144
			ps241.OverlayValues[145] = d145
			ps241.OverlayValues[146] = d146
			ps241.OverlayValues[147] = d147
			ps241.OverlayValues[148] = d148
			ps241.OverlayValues[149] = d149
			ps241.OverlayValues[150] = d150
			ps241.OverlayValues[151] = d151
			ps241.OverlayValues[152] = d152
			ps241.OverlayValues[153] = d153
			ps241.OverlayValues[154] = d154
			ps241.OverlayValues[155] = d155
			ps241.OverlayValues[156] = d156
			ps241.OverlayValues[157] = d157
			ps241.OverlayValues[158] = d158
			ps241.OverlayValues[159] = d159
			ps241.OverlayValues[160] = d160
			ps241.OverlayValues[161] = d161
			ps241.OverlayValues[162] = d162
			ps241.OverlayValues[163] = d163
			ps241.OverlayValues[164] = d164
			ps241.OverlayValues[165] = d165
			ps241.OverlayValues[166] = d166
			ps241.OverlayValues[167] = d167
			ps241.OverlayValues[168] = d168
			ps241.OverlayValues[169] = d169
			ps241.OverlayValues[170] = d170
			ps241.OverlayValues[171] = d171
			ps241.OverlayValues[172] = d172
			ps241.OverlayValues[173] = d173
			ps241.OverlayValues[174] = d174
			ps241.OverlayValues[175] = d175
			ps241.OverlayValues[176] = d176
			ps241.OverlayValues[177] = d177
			ps241.OverlayValues[178] = d178
			ps241.OverlayValues[179] = d179
			ps241.OverlayValues[180] = d180
			ps241.OverlayValues[181] = d181
			ps241.OverlayValues[182] = d182
			ps241.OverlayValues[183] = d183
			ps241.OverlayValues[184] = d184
			ps241.OverlayValues[185] = d185
			ps241.OverlayValues[186] = d186
			ps241.OverlayValues[187] = d187
			ps241.OverlayValues[188] = d188
			ps241.OverlayValues[189] = d189
			ps241.OverlayValues[190] = d190
			ps241.OverlayValues[191] = d191
			ps241.OverlayValues[192] = d192
			ps241.OverlayValues[193] = d193
			ps241.OverlayValues[194] = d194
			ps241.OverlayValues[195] = d195
			ps241.OverlayValues[196] = d196
			ps241.OverlayValues[197] = d197
			ps241.OverlayValues[198] = d198
			ps241.OverlayValues[199] = d199
			ps241.OverlayValues[200] = d200
			ps241.OverlayValues[201] = d201
			ps241.OverlayValues[202] = d202
			ps241.OverlayValues[203] = d203
			ps241.OverlayValues[204] = d204
			ps241.OverlayValues[205] = d205
			ps241.OverlayValues[206] = d206
			ps241.OverlayValues[207] = d207
			ps241.OverlayValues[208] = d208
			ps241.OverlayValues[209] = d209
			ps241.OverlayValues[210] = d210
			ps241.OverlayValues[211] = d211
			ps241.OverlayValues[212] = d212
			ps241.OverlayValues[213] = d213
			ps241.OverlayValues[214] = d214
			ps241.OverlayValues[215] = d215
			ps241.OverlayValues[216] = d216
			ps241.OverlayValues[217] = d217
			ps241.OverlayValues[218] = d218
			ps241.OverlayValues[219] = d219
			ps241.OverlayValues[220] = d220
			ps241.OverlayValues[221] = d221
			ps241.OverlayValues[222] = d222
			ps241.OverlayValues[223] = d223
			ps241.OverlayValues[224] = d224
			ps241.OverlayValues[225] = d225
			ps241.OverlayValues[226] = d226
			ps241.OverlayValues[227] = d227
			ps241.OverlayValues[228] = d228
			ps241.OverlayValues[229] = d229
			ps241.OverlayValues[230] = d230
			ps241.OverlayValues[231] = d231
			ps241.OverlayValues[232] = d232
			ps241.OverlayValues[233] = d233
			ps241.OverlayValues[234] = d234
			ps241.OverlayValues[235] = d235
			ps241.OverlayValues[236] = d236
			ps241.OverlayValues[237] = d237
			ps241.OverlayValues[238] = d238
			ps241.OverlayValues[239] = d239
			ps241.OverlayValues[240] = d240
					return bbs[1].RenderPS(ps241)
				}
			ps242 := scm.PhiState{General: ps.General}
			ps242.OverlayValues = make([]scm.JITValueDesc, 241)
			ps242.OverlayValues[0] = d0
			ps242.OverlayValues[1] = d1
			ps242.OverlayValues[2] = d2
			ps242.OverlayValues[3] = d3
			ps242.OverlayValues[4] = d4
			ps242.OverlayValues[5] = d5
			ps242.OverlayValues[6] = d6
			ps242.OverlayValues[7] = d7
			ps242.OverlayValues[8] = d8
			ps242.OverlayValues[9] = d9
			ps242.OverlayValues[10] = d10
			ps242.OverlayValues[11] = d11
			ps242.OverlayValues[12] = d12
			ps242.OverlayValues[13] = d13
			ps242.OverlayValues[14] = d14
			ps242.OverlayValues[15] = d15
			ps242.OverlayValues[16] = d16
			ps242.OverlayValues[17] = d17
			ps242.OverlayValues[18] = d18
			ps242.OverlayValues[19] = d19
			ps242.OverlayValues[20] = d20
			ps242.OverlayValues[21] = d21
			ps242.OverlayValues[22] = d22
			ps242.OverlayValues[23] = d23
			ps242.OverlayValues[24] = d24
			ps242.OverlayValues[25] = d25
			ps242.OverlayValues[26] = d26
			ps242.OverlayValues[27] = d27
			ps242.OverlayValues[28] = d28
			ps242.OverlayValues[29] = d29
			ps242.OverlayValues[30] = d30
			ps242.OverlayValues[31] = d31
			ps242.OverlayValues[32] = d32
			ps242.OverlayValues[33] = d33
			ps242.OverlayValues[34] = d34
			ps242.OverlayValues[35] = d35
			ps242.OverlayValues[36] = d36
			ps242.OverlayValues[37] = d37
			ps242.OverlayValues[38] = d38
			ps242.OverlayValues[39] = d39
			ps242.OverlayValues[40] = d40
			ps242.OverlayValues[41] = d41
			ps242.OverlayValues[42] = d42
			ps242.OverlayValues[43] = d43
			ps242.OverlayValues[44] = d44
			ps242.OverlayValues[45] = d45
			ps242.OverlayValues[46] = d46
			ps242.OverlayValues[47] = d47
			ps242.OverlayValues[48] = d48
			ps242.OverlayValues[49] = d49
			ps242.OverlayValues[50] = d50
			ps242.OverlayValues[51] = d51
			ps242.OverlayValues[52] = d52
			ps242.OverlayValues[53] = d53
			ps242.OverlayValues[54] = d54
			ps242.OverlayValues[55] = d55
			ps242.OverlayValues[56] = d56
			ps242.OverlayValues[57] = d57
			ps242.OverlayValues[58] = d58
			ps242.OverlayValues[59] = d59
			ps242.OverlayValues[60] = d60
			ps242.OverlayValues[61] = d61
			ps242.OverlayValues[62] = d62
			ps242.OverlayValues[63] = d63
			ps242.OverlayValues[64] = d64
			ps242.OverlayValues[65] = d65
			ps242.OverlayValues[66] = d66
			ps242.OverlayValues[67] = d67
			ps242.OverlayValues[68] = d68
			ps242.OverlayValues[69] = d69
			ps242.OverlayValues[70] = d70
			ps242.OverlayValues[71] = d71
			ps242.OverlayValues[72] = d72
			ps242.OverlayValues[73] = d73
			ps242.OverlayValues[74] = d74
			ps242.OverlayValues[75] = d75
			ps242.OverlayValues[76] = d76
			ps242.OverlayValues[77] = d77
			ps242.OverlayValues[78] = d78
			ps242.OverlayValues[79] = d79
			ps242.OverlayValues[80] = d80
			ps242.OverlayValues[81] = d81
			ps242.OverlayValues[82] = d82
			ps242.OverlayValues[83] = d83
			ps242.OverlayValues[84] = d84
			ps242.OverlayValues[85] = d85
			ps242.OverlayValues[86] = d86
			ps242.OverlayValues[87] = d87
			ps242.OverlayValues[88] = d88
			ps242.OverlayValues[89] = d89
			ps242.OverlayValues[90] = d90
			ps242.OverlayValues[91] = d91
			ps242.OverlayValues[92] = d92
			ps242.OverlayValues[93] = d93
			ps242.OverlayValues[94] = d94
			ps242.OverlayValues[95] = d95
			ps242.OverlayValues[96] = d96
			ps242.OverlayValues[97] = d97
			ps242.OverlayValues[98] = d98
			ps242.OverlayValues[99] = d99
			ps242.OverlayValues[100] = d100
			ps242.OverlayValues[101] = d101
			ps242.OverlayValues[102] = d102
			ps242.OverlayValues[103] = d103
			ps242.OverlayValues[104] = d104
			ps242.OverlayValues[105] = d105
			ps242.OverlayValues[106] = d106
			ps242.OverlayValues[107] = d107
			ps242.OverlayValues[108] = d108
			ps242.OverlayValues[109] = d109
			ps242.OverlayValues[110] = d110
			ps242.OverlayValues[111] = d111
			ps242.OverlayValues[112] = d112
			ps242.OverlayValues[113] = d113
			ps242.OverlayValues[114] = d114
			ps242.OverlayValues[115] = d115
			ps242.OverlayValues[116] = d116
			ps242.OverlayValues[117] = d117
			ps242.OverlayValues[118] = d118
			ps242.OverlayValues[119] = d119
			ps242.OverlayValues[120] = d120
			ps242.OverlayValues[121] = d121
			ps242.OverlayValues[122] = d122
			ps242.OverlayValues[123] = d123
			ps242.OverlayValues[124] = d124
			ps242.OverlayValues[125] = d125
			ps242.OverlayValues[126] = d126
			ps242.OverlayValues[127] = d127
			ps242.OverlayValues[128] = d128
			ps242.OverlayValues[129] = d129
			ps242.OverlayValues[130] = d130
			ps242.OverlayValues[131] = d131
			ps242.OverlayValues[132] = d132
			ps242.OverlayValues[133] = d133
			ps242.OverlayValues[134] = d134
			ps242.OverlayValues[135] = d135
			ps242.OverlayValues[136] = d136
			ps242.OverlayValues[137] = d137
			ps242.OverlayValues[138] = d138
			ps242.OverlayValues[139] = d139
			ps242.OverlayValues[140] = d140
			ps242.OverlayValues[141] = d141
			ps242.OverlayValues[142] = d142
			ps242.OverlayValues[143] = d143
			ps242.OverlayValues[144] = d144
			ps242.OverlayValues[145] = d145
			ps242.OverlayValues[146] = d146
			ps242.OverlayValues[147] = d147
			ps242.OverlayValues[148] = d148
			ps242.OverlayValues[149] = d149
			ps242.OverlayValues[150] = d150
			ps242.OverlayValues[151] = d151
			ps242.OverlayValues[152] = d152
			ps242.OverlayValues[153] = d153
			ps242.OverlayValues[154] = d154
			ps242.OverlayValues[155] = d155
			ps242.OverlayValues[156] = d156
			ps242.OverlayValues[157] = d157
			ps242.OverlayValues[158] = d158
			ps242.OverlayValues[159] = d159
			ps242.OverlayValues[160] = d160
			ps242.OverlayValues[161] = d161
			ps242.OverlayValues[162] = d162
			ps242.OverlayValues[163] = d163
			ps242.OverlayValues[164] = d164
			ps242.OverlayValues[165] = d165
			ps242.OverlayValues[166] = d166
			ps242.OverlayValues[167] = d167
			ps242.OverlayValues[168] = d168
			ps242.OverlayValues[169] = d169
			ps242.OverlayValues[170] = d170
			ps242.OverlayValues[171] = d171
			ps242.OverlayValues[172] = d172
			ps242.OverlayValues[173] = d173
			ps242.OverlayValues[174] = d174
			ps242.OverlayValues[175] = d175
			ps242.OverlayValues[176] = d176
			ps242.OverlayValues[177] = d177
			ps242.OverlayValues[178] = d178
			ps242.OverlayValues[179] = d179
			ps242.OverlayValues[180] = d180
			ps242.OverlayValues[181] = d181
			ps242.OverlayValues[182] = d182
			ps242.OverlayValues[183] = d183
			ps242.OverlayValues[184] = d184
			ps242.OverlayValues[185] = d185
			ps242.OverlayValues[186] = d186
			ps242.OverlayValues[187] = d187
			ps242.OverlayValues[188] = d188
			ps242.OverlayValues[189] = d189
			ps242.OverlayValues[190] = d190
			ps242.OverlayValues[191] = d191
			ps242.OverlayValues[192] = d192
			ps242.OverlayValues[193] = d193
			ps242.OverlayValues[194] = d194
			ps242.OverlayValues[195] = d195
			ps242.OverlayValues[196] = d196
			ps242.OverlayValues[197] = d197
			ps242.OverlayValues[198] = d198
			ps242.OverlayValues[199] = d199
			ps242.OverlayValues[200] = d200
			ps242.OverlayValues[201] = d201
			ps242.OverlayValues[202] = d202
			ps242.OverlayValues[203] = d203
			ps242.OverlayValues[204] = d204
			ps242.OverlayValues[205] = d205
			ps242.OverlayValues[206] = d206
			ps242.OverlayValues[207] = d207
			ps242.OverlayValues[208] = d208
			ps242.OverlayValues[209] = d209
			ps242.OverlayValues[210] = d210
			ps242.OverlayValues[211] = d211
			ps242.OverlayValues[212] = d212
			ps242.OverlayValues[213] = d213
			ps242.OverlayValues[214] = d214
			ps242.OverlayValues[215] = d215
			ps242.OverlayValues[216] = d216
			ps242.OverlayValues[217] = d217
			ps242.OverlayValues[218] = d218
			ps242.OverlayValues[219] = d219
			ps242.OverlayValues[220] = d220
			ps242.OverlayValues[221] = d221
			ps242.OverlayValues[222] = d222
			ps242.OverlayValues[223] = d223
			ps242.OverlayValues[224] = d224
			ps242.OverlayValues[225] = d225
			ps242.OverlayValues[226] = d226
			ps242.OverlayValues[227] = d227
			ps242.OverlayValues[228] = d228
			ps242.OverlayValues[229] = d229
			ps242.OverlayValues[230] = d230
			ps242.OverlayValues[231] = d231
			ps242.OverlayValues[232] = d232
			ps242.OverlayValues[233] = d233
			ps242.OverlayValues[234] = d234
			ps242.OverlayValues[235] = d235
			ps242.OverlayValues[236] = d236
			ps242.OverlayValues[237] = d237
			ps242.OverlayValues[238] = d238
			ps242.OverlayValues[239] = d239
			ps242.OverlayValues[240] = d240
				return bbs[2].RenderPS(ps242)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl68 := ctx.ReserveLabel()
			lbl69 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d240.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl68)
			ctx.EmitJmp(lbl69)
			ctx.MarkLabel(lbl68)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl69)
			ctx.EmitJmp(lbl3)
			ps243 := scm.PhiState{General: true}
			ps243.OverlayValues = make([]scm.JITValueDesc, 241)
			ps243.OverlayValues[0] = d0
			ps243.OverlayValues[1] = d1
			ps243.OverlayValues[2] = d2
			ps243.OverlayValues[3] = d3
			ps243.OverlayValues[4] = d4
			ps243.OverlayValues[5] = d5
			ps243.OverlayValues[6] = d6
			ps243.OverlayValues[7] = d7
			ps243.OverlayValues[8] = d8
			ps243.OverlayValues[9] = d9
			ps243.OverlayValues[10] = d10
			ps243.OverlayValues[11] = d11
			ps243.OverlayValues[12] = d12
			ps243.OverlayValues[13] = d13
			ps243.OverlayValues[14] = d14
			ps243.OverlayValues[15] = d15
			ps243.OverlayValues[16] = d16
			ps243.OverlayValues[17] = d17
			ps243.OverlayValues[18] = d18
			ps243.OverlayValues[19] = d19
			ps243.OverlayValues[20] = d20
			ps243.OverlayValues[21] = d21
			ps243.OverlayValues[22] = d22
			ps243.OverlayValues[23] = d23
			ps243.OverlayValues[24] = d24
			ps243.OverlayValues[25] = d25
			ps243.OverlayValues[26] = d26
			ps243.OverlayValues[27] = d27
			ps243.OverlayValues[28] = d28
			ps243.OverlayValues[29] = d29
			ps243.OverlayValues[30] = d30
			ps243.OverlayValues[31] = d31
			ps243.OverlayValues[32] = d32
			ps243.OverlayValues[33] = d33
			ps243.OverlayValues[34] = d34
			ps243.OverlayValues[35] = d35
			ps243.OverlayValues[36] = d36
			ps243.OverlayValues[37] = d37
			ps243.OverlayValues[38] = d38
			ps243.OverlayValues[39] = d39
			ps243.OverlayValues[40] = d40
			ps243.OverlayValues[41] = d41
			ps243.OverlayValues[42] = d42
			ps243.OverlayValues[43] = d43
			ps243.OverlayValues[44] = d44
			ps243.OverlayValues[45] = d45
			ps243.OverlayValues[46] = d46
			ps243.OverlayValues[47] = d47
			ps243.OverlayValues[48] = d48
			ps243.OverlayValues[49] = d49
			ps243.OverlayValues[50] = d50
			ps243.OverlayValues[51] = d51
			ps243.OverlayValues[52] = d52
			ps243.OverlayValues[53] = d53
			ps243.OverlayValues[54] = d54
			ps243.OverlayValues[55] = d55
			ps243.OverlayValues[56] = d56
			ps243.OverlayValues[57] = d57
			ps243.OverlayValues[58] = d58
			ps243.OverlayValues[59] = d59
			ps243.OverlayValues[60] = d60
			ps243.OverlayValues[61] = d61
			ps243.OverlayValues[62] = d62
			ps243.OverlayValues[63] = d63
			ps243.OverlayValues[64] = d64
			ps243.OverlayValues[65] = d65
			ps243.OverlayValues[66] = d66
			ps243.OverlayValues[67] = d67
			ps243.OverlayValues[68] = d68
			ps243.OverlayValues[69] = d69
			ps243.OverlayValues[70] = d70
			ps243.OverlayValues[71] = d71
			ps243.OverlayValues[72] = d72
			ps243.OverlayValues[73] = d73
			ps243.OverlayValues[74] = d74
			ps243.OverlayValues[75] = d75
			ps243.OverlayValues[76] = d76
			ps243.OverlayValues[77] = d77
			ps243.OverlayValues[78] = d78
			ps243.OverlayValues[79] = d79
			ps243.OverlayValues[80] = d80
			ps243.OverlayValues[81] = d81
			ps243.OverlayValues[82] = d82
			ps243.OverlayValues[83] = d83
			ps243.OverlayValues[84] = d84
			ps243.OverlayValues[85] = d85
			ps243.OverlayValues[86] = d86
			ps243.OverlayValues[87] = d87
			ps243.OverlayValues[88] = d88
			ps243.OverlayValues[89] = d89
			ps243.OverlayValues[90] = d90
			ps243.OverlayValues[91] = d91
			ps243.OverlayValues[92] = d92
			ps243.OverlayValues[93] = d93
			ps243.OverlayValues[94] = d94
			ps243.OverlayValues[95] = d95
			ps243.OverlayValues[96] = d96
			ps243.OverlayValues[97] = d97
			ps243.OverlayValues[98] = d98
			ps243.OverlayValues[99] = d99
			ps243.OverlayValues[100] = d100
			ps243.OverlayValues[101] = d101
			ps243.OverlayValues[102] = d102
			ps243.OverlayValues[103] = d103
			ps243.OverlayValues[104] = d104
			ps243.OverlayValues[105] = d105
			ps243.OverlayValues[106] = d106
			ps243.OverlayValues[107] = d107
			ps243.OverlayValues[108] = d108
			ps243.OverlayValues[109] = d109
			ps243.OverlayValues[110] = d110
			ps243.OverlayValues[111] = d111
			ps243.OverlayValues[112] = d112
			ps243.OverlayValues[113] = d113
			ps243.OverlayValues[114] = d114
			ps243.OverlayValues[115] = d115
			ps243.OverlayValues[116] = d116
			ps243.OverlayValues[117] = d117
			ps243.OverlayValues[118] = d118
			ps243.OverlayValues[119] = d119
			ps243.OverlayValues[120] = d120
			ps243.OverlayValues[121] = d121
			ps243.OverlayValues[122] = d122
			ps243.OverlayValues[123] = d123
			ps243.OverlayValues[124] = d124
			ps243.OverlayValues[125] = d125
			ps243.OverlayValues[126] = d126
			ps243.OverlayValues[127] = d127
			ps243.OverlayValues[128] = d128
			ps243.OverlayValues[129] = d129
			ps243.OverlayValues[130] = d130
			ps243.OverlayValues[131] = d131
			ps243.OverlayValues[132] = d132
			ps243.OverlayValues[133] = d133
			ps243.OverlayValues[134] = d134
			ps243.OverlayValues[135] = d135
			ps243.OverlayValues[136] = d136
			ps243.OverlayValues[137] = d137
			ps243.OverlayValues[138] = d138
			ps243.OverlayValues[139] = d139
			ps243.OverlayValues[140] = d140
			ps243.OverlayValues[141] = d141
			ps243.OverlayValues[142] = d142
			ps243.OverlayValues[143] = d143
			ps243.OverlayValues[144] = d144
			ps243.OverlayValues[145] = d145
			ps243.OverlayValues[146] = d146
			ps243.OverlayValues[147] = d147
			ps243.OverlayValues[148] = d148
			ps243.OverlayValues[149] = d149
			ps243.OverlayValues[150] = d150
			ps243.OverlayValues[151] = d151
			ps243.OverlayValues[152] = d152
			ps243.OverlayValues[153] = d153
			ps243.OverlayValues[154] = d154
			ps243.OverlayValues[155] = d155
			ps243.OverlayValues[156] = d156
			ps243.OverlayValues[157] = d157
			ps243.OverlayValues[158] = d158
			ps243.OverlayValues[159] = d159
			ps243.OverlayValues[160] = d160
			ps243.OverlayValues[161] = d161
			ps243.OverlayValues[162] = d162
			ps243.OverlayValues[163] = d163
			ps243.OverlayValues[164] = d164
			ps243.OverlayValues[165] = d165
			ps243.OverlayValues[166] = d166
			ps243.OverlayValues[167] = d167
			ps243.OverlayValues[168] = d168
			ps243.OverlayValues[169] = d169
			ps243.OverlayValues[170] = d170
			ps243.OverlayValues[171] = d171
			ps243.OverlayValues[172] = d172
			ps243.OverlayValues[173] = d173
			ps243.OverlayValues[174] = d174
			ps243.OverlayValues[175] = d175
			ps243.OverlayValues[176] = d176
			ps243.OverlayValues[177] = d177
			ps243.OverlayValues[178] = d178
			ps243.OverlayValues[179] = d179
			ps243.OverlayValues[180] = d180
			ps243.OverlayValues[181] = d181
			ps243.OverlayValues[182] = d182
			ps243.OverlayValues[183] = d183
			ps243.OverlayValues[184] = d184
			ps243.OverlayValues[185] = d185
			ps243.OverlayValues[186] = d186
			ps243.OverlayValues[187] = d187
			ps243.OverlayValues[188] = d188
			ps243.OverlayValues[189] = d189
			ps243.OverlayValues[190] = d190
			ps243.OverlayValues[191] = d191
			ps243.OverlayValues[192] = d192
			ps243.OverlayValues[193] = d193
			ps243.OverlayValues[194] = d194
			ps243.OverlayValues[195] = d195
			ps243.OverlayValues[196] = d196
			ps243.OverlayValues[197] = d197
			ps243.OverlayValues[198] = d198
			ps243.OverlayValues[199] = d199
			ps243.OverlayValues[200] = d200
			ps243.OverlayValues[201] = d201
			ps243.OverlayValues[202] = d202
			ps243.OverlayValues[203] = d203
			ps243.OverlayValues[204] = d204
			ps243.OverlayValues[205] = d205
			ps243.OverlayValues[206] = d206
			ps243.OverlayValues[207] = d207
			ps243.OverlayValues[208] = d208
			ps243.OverlayValues[209] = d209
			ps243.OverlayValues[210] = d210
			ps243.OverlayValues[211] = d211
			ps243.OverlayValues[212] = d212
			ps243.OverlayValues[213] = d213
			ps243.OverlayValues[214] = d214
			ps243.OverlayValues[215] = d215
			ps243.OverlayValues[216] = d216
			ps243.OverlayValues[217] = d217
			ps243.OverlayValues[218] = d218
			ps243.OverlayValues[219] = d219
			ps243.OverlayValues[220] = d220
			ps243.OverlayValues[221] = d221
			ps243.OverlayValues[222] = d222
			ps243.OverlayValues[223] = d223
			ps243.OverlayValues[224] = d224
			ps243.OverlayValues[225] = d225
			ps243.OverlayValues[226] = d226
			ps243.OverlayValues[227] = d227
			ps243.OverlayValues[228] = d228
			ps243.OverlayValues[229] = d229
			ps243.OverlayValues[230] = d230
			ps243.OverlayValues[231] = d231
			ps243.OverlayValues[232] = d232
			ps243.OverlayValues[233] = d233
			ps243.OverlayValues[234] = d234
			ps243.OverlayValues[235] = d235
			ps243.OverlayValues[236] = d236
			ps243.OverlayValues[237] = d237
			ps243.OverlayValues[238] = d238
			ps243.OverlayValues[239] = d239
			ps243.OverlayValues[240] = d240
			ps244 := scm.PhiState{General: true}
			ps244.OverlayValues = make([]scm.JITValueDesc, 241)
			ps244.OverlayValues[0] = d0
			ps244.OverlayValues[1] = d1
			ps244.OverlayValues[2] = d2
			ps244.OverlayValues[3] = d3
			ps244.OverlayValues[4] = d4
			ps244.OverlayValues[5] = d5
			ps244.OverlayValues[6] = d6
			ps244.OverlayValues[7] = d7
			ps244.OverlayValues[8] = d8
			ps244.OverlayValues[9] = d9
			ps244.OverlayValues[10] = d10
			ps244.OverlayValues[11] = d11
			ps244.OverlayValues[12] = d12
			ps244.OverlayValues[13] = d13
			ps244.OverlayValues[14] = d14
			ps244.OverlayValues[15] = d15
			ps244.OverlayValues[16] = d16
			ps244.OverlayValues[17] = d17
			ps244.OverlayValues[18] = d18
			ps244.OverlayValues[19] = d19
			ps244.OverlayValues[20] = d20
			ps244.OverlayValues[21] = d21
			ps244.OverlayValues[22] = d22
			ps244.OverlayValues[23] = d23
			ps244.OverlayValues[24] = d24
			ps244.OverlayValues[25] = d25
			ps244.OverlayValues[26] = d26
			ps244.OverlayValues[27] = d27
			ps244.OverlayValues[28] = d28
			ps244.OverlayValues[29] = d29
			ps244.OverlayValues[30] = d30
			ps244.OverlayValues[31] = d31
			ps244.OverlayValues[32] = d32
			ps244.OverlayValues[33] = d33
			ps244.OverlayValues[34] = d34
			ps244.OverlayValues[35] = d35
			ps244.OverlayValues[36] = d36
			ps244.OverlayValues[37] = d37
			ps244.OverlayValues[38] = d38
			ps244.OverlayValues[39] = d39
			ps244.OverlayValues[40] = d40
			ps244.OverlayValues[41] = d41
			ps244.OverlayValues[42] = d42
			ps244.OverlayValues[43] = d43
			ps244.OverlayValues[44] = d44
			ps244.OverlayValues[45] = d45
			ps244.OverlayValues[46] = d46
			ps244.OverlayValues[47] = d47
			ps244.OverlayValues[48] = d48
			ps244.OverlayValues[49] = d49
			ps244.OverlayValues[50] = d50
			ps244.OverlayValues[51] = d51
			ps244.OverlayValues[52] = d52
			ps244.OverlayValues[53] = d53
			ps244.OverlayValues[54] = d54
			ps244.OverlayValues[55] = d55
			ps244.OverlayValues[56] = d56
			ps244.OverlayValues[57] = d57
			ps244.OverlayValues[58] = d58
			ps244.OverlayValues[59] = d59
			ps244.OverlayValues[60] = d60
			ps244.OverlayValues[61] = d61
			ps244.OverlayValues[62] = d62
			ps244.OverlayValues[63] = d63
			ps244.OverlayValues[64] = d64
			ps244.OverlayValues[65] = d65
			ps244.OverlayValues[66] = d66
			ps244.OverlayValues[67] = d67
			ps244.OverlayValues[68] = d68
			ps244.OverlayValues[69] = d69
			ps244.OverlayValues[70] = d70
			ps244.OverlayValues[71] = d71
			ps244.OverlayValues[72] = d72
			ps244.OverlayValues[73] = d73
			ps244.OverlayValues[74] = d74
			ps244.OverlayValues[75] = d75
			ps244.OverlayValues[76] = d76
			ps244.OverlayValues[77] = d77
			ps244.OverlayValues[78] = d78
			ps244.OverlayValues[79] = d79
			ps244.OverlayValues[80] = d80
			ps244.OverlayValues[81] = d81
			ps244.OverlayValues[82] = d82
			ps244.OverlayValues[83] = d83
			ps244.OverlayValues[84] = d84
			ps244.OverlayValues[85] = d85
			ps244.OverlayValues[86] = d86
			ps244.OverlayValues[87] = d87
			ps244.OverlayValues[88] = d88
			ps244.OverlayValues[89] = d89
			ps244.OverlayValues[90] = d90
			ps244.OverlayValues[91] = d91
			ps244.OverlayValues[92] = d92
			ps244.OverlayValues[93] = d93
			ps244.OverlayValues[94] = d94
			ps244.OverlayValues[95] = d95
			ps244.OverlayValues[96] = d96
			ps244.OverlayValues[97] = d97
			ps244.OverlayValues[98] = d98
			ps244.OverlayValues[99] = d99
			ps244.OverlayValues[100] = d100
			ps244.OverlayValues[101] = d101
			ps244.OverlayValues[102] = d102
			ps244.OverlayValues[103] = d103
			ps244.OverlayValues[104] = d104
			ps244.OverlayValues[105] = d105
			ps244.OverlayValues[106] = d106
			ps244.OverlayValues[107] = d107
			ps244.OverlayValues[108] = d108
			ps244.OverlayValues[109] = d109
			ps244.OverlayValues[110] = d110
			ps244.OverlayValues[111] = d111
			ps244.OverlayValues[112] = d112
			ps244.OverlayValues[113] = d113
			ps244.OverlayValues[114] = d114
			ps244.OverlayValues[115] = d115
			ps244.OverlayValues[116] = d116
			ps244.OverlayValues[117] = d117
			ps244.OverlayValues[118] = d118
			ps244.OverlayValues[119] = d119
			ps244.OverlayValues[120] = d120
			ps244.OverlayValues[121] = d121
			ps244.OverlayValues[122] = d122
			ps244.OverlayValues[123] = d123
			ps244.OverlayValues[124] = d124
			ps244.OverlayValues[125] = d125
			ps244.OverlayValues[126] = d126
			ps244.OverlayValues[127] = d127
			ps244.OverlayValues[128] = d128
			ps244.OverlayValues[129] = d129
			ps244.OverlayValues[130] = d130
			ps244.OverlayValues[131] = d131
			ps244.OverlayValues[132] = d132
			ps244.OverlayValues[133] = d133
			ps244.OverlayValues[134] = d134
			ps244.OverlayValues[135] = d135
			ps244.OverlayValues[136] = d136
			ps244.OverlayValues[137] = d137
			ps244.OverlayValues[138] = d138
			ps244.OverlayValues[139] = d139
			ps244.OverlayValues[140] = d140
			ps244.OverlayValues[141] = d141
			ps244.OverlayValues[142] = d142
			ps244.OverlayValues[143] = d143
			ps244.OverlayValues[144] = d144
			ps244.OverlayValues[145] = d145
			ps244.OverlayValues[146] = d146
			ps244.OverlayValues[147] = d147
			ps244.OverlayValues[148] = d148
			ps244.OverlayValues[149] = d149
			ps244.OverlayValues[150] = d150
			ps244.OverlayValues[151] = d151
			ps244.OverlayValues[152] = d152
			ps244.OverlayValues[153] = d153
			ps244.OverlayValues[154] = d154
			ps244.OverlayValues[155] = d155
			ps244.OverlayValues[156] = d156
			ps244.OverlayValues[157] = d157
			ps244.OverlayValues[158] = d158
			ps244.OverlayValues[159] = d159
			ps244.OverlayValues[160] = d160
			ps244.OverlayValues[161] = d161
			ps244.OverlayValues[162] = d162
			ps244.OverlayValues[163] = d163
			ps244.OverlayValues[164] = d164
			ps244.OverlayValues[165] = d165
			ps244.OverlayValues[166] = d166
			ps244.OverlayValues[167] = d167
			ps244.OverlayValues[168] = d168
			ps244.OverlayValues[169] = d169
			ps244.OverlayValues[170] = d170
			ps244.OverlayValues[171] = d171
			ps244.OverlayValues[172] = d172
			ps244.OverlayValues[173] = d173
			ps244.OverlayValues[174] = d174
			ps244.OverlayValues[175] = d175
			ps244.OverlayValues[176] = d176
			ps244.OverlayValues[177] = d177
			ps244.OverlayValues[178] = d178
			ps244.OverlayValues[179] = d179
			ps244.OverlayValues[180] = d180
			ps244.OverlayValues[181] = d181
			ps244.OverlayValues[182] = d182
			ps244.OverlayValues[183] = d183
			ps244.OverlayValues[184] = d184
			ps244.OverlayValues[185] = d185
			ps244.OverlayValues[186] = d186
			ps244.OverlayValues[187] = d187
			ps244.OverlayValues[188] = d188
			ps244.OverlayValues[189] = d189
			ps244.OverlayValues[190] = d190
			ps244.OverlayValues[191] = d191
			ps244.OverlayValues[192] = d192
			ps244.OverlayValues[193] = d193
			ps244.OverlayValues[194] = d194
			ps244.OverlayValues[195] = d195
			ps244.OverlayValues[196] = d196
			ps244.OverlayValues[197] = d197
			ps244.OverlayValues[198] = d198
			ps244.OverlayValues[199] = d199
			ps244.OverlayValues[200] = d200
			ps244.OverlayValues[201] = d201
			ps244.OverlayValues[202] = d202
			ps244.OverlayValues[203] = d203
			ps244.OverlayValues[204] = d204
			ps244.OverlayValues[205] = d205
			ps244.OverlayValues[206] = d206
			ps244.OverlayValues[207] = d207
			ps244.OverlayValues[208] = d208
			ps244.OverlayValues[209] = d209
			ps244.OverlayValues[210] = d210
			ps244.OverlayValues[211] = d211
			ps244.OverlayValues[212] = d212
			ps244.OverlayValues[213] = d213
			ps244.OverlayValues[214] = d214
			ps244.OverlayValues[215] = d215
			ps244.OverlayValues[216] = d216
			ps244.OverlayValues[217] = d217
			ps244.OverlayValues[218] = d218
			ps244.OverlayValues[219] = d219
			ps244.OverlayValues[220] = d220
			ps244.OverlayValues[221] = d221
			ps244.OverlayValues[222] = d222
			ps244.OverlayValues[223] = d223
			ps244.OverlayValues[224] = d224
			ps244.OverlayValues[225] = d225
			ps244.OverlayValues[226] = d226
			ps244.OverlayValues[227] = d227
			ps244.OverlayValues[228] = d228
			ps244.OverlayValues[229] = d229
			ps244.OverlayValues[230] = d230
			ps244.OverlayValues[231] = d231
			ps244.OverlayValues[232] = d232
			ps244.OverlayValues[233] = d233
			ps244.OverlayValues[234] = d234
			ps244.OverlayValues[235] = d235
			ps244.OverlayValues[236] = d236
			ps244.OverlayValues[237] = d237
			ps244.OverlayValues[238] = d238
			ps244.OverlayValues[239] = d239
			ps244.OverlayValues[240] = d240
			snap245 := d0
			snap246 := d1
			snap247 := d2
			snap248 := d3
			snap249 := d4
			snap250 := d5
			snap251 := d6
			snap252 := d7
			snap253 := d8
			snap254 := d9
			snap255 := d10
			snap256 := d11
			snap257 := d12
			snap258 := d13
			snap259 := d14
			snap260 := d15
			snap261 := d16
			snap262 := d17
			snap263 := d18
			snap264 := d19
			snap265 := d20
			snap266 := d21
			snap267 := d22
			snap268 := d23
			snap269 := d24
			snap270 := d25
			snap271 := d26
			snap272 := d27
			snap273 := d28
			snap274 := d29
			snap275 := d30
			snap276 := d31
			snap277 := d32
			snap278 := d33
			snap279 := d34
			snap280 := d35
			snap281 := d36
			snap282 := d37
			snap283 := d38
			snap284 := d39
			snap285 := d40
			snap286 := d41
			snap287 := d42
			snap288 := d43
			snap289 := d44
			snap290 := d45
			snap291 := d46
			snap292 := d47
			snap293 := d48
			snap294 := d49
			snap295 := d50
			snap296 := d51
			snap297 := d52
			snap298 := d53
			snap299 := d54
			snap300 := d55
			snap301 := d56
			snap302 := d57
			snap303 := d58
			snap304 := d59
			snap305 := d60
			snap306 := d61
			snap307 := d62
			snap308 := d63
			snap309 := d64
			snap310 := d65
			snap311 := d66
			snap312 := d67
			snap313 := d68
			snap314 := d69
			snap315 := d70
			snap316 := d71
			snap317 := d72
			snap318 := d73
			snap319 := d74
			snap320 := d75
			snap321 := d76
			snap322 := d77
			snap323 := d78
			snap324 := d79
			snap325 := d80
			snap326 := d81
			snap327 := d82
			snap328 := d83
			snap329 := d84
			snap330 := d85
			snap331 := d86
			snap332 := d87
			snap333 := d88
			snap334 := d89
			snap335 := d90
			snap336 := d91
			snap337 := d92
			snap338 := d93
			snap339 := d94
			snap340 := d95
			snap341 := d96
			snap342 := d97
			snap343 := d98
			snap344 := d99
			snap345 := d100
			snap346 := d101
			snap347 := d102
			snap348 := d103
			snap349 := d104
			snap350 := d105
			snap351 := d106
			snap352 := d107
			snap353 := d108
			snap354 := d109
			snap355 := d110
			snap356 := d111
			snap357 := d112
			snap358 := d113
			snap359 := d114
			snap360 := d115
			snap361 := d116
			snap362 := d117
			snap363 := d118
			snap364 := d119
			snap365 := d120
			snap366 := d121
			snap367 := d122
			snap368 := d123
			snap369 := d124
			snap370 := d125
			snap371 := d126
			snap372 := d127
			snap373 := d128
			snap374 := d129
			snap375 := d130
			snap376 := d131
			snap377 := d132
			snap378 := d133
			snap379 := d134
			snap380 := d135
			snap381 := d136
			snap382 := d137
			snap383 := d138
			snap384 := d139
			snap385 := d140
			snap386 := d141
			snap387 := d142
			snap388 := d143
			snap389 := d144
			snap390 := d145
			snap391 := d146
			snap392 := d147
			snap393 := d148
			snap394 := d149
			snap395 := d150
			snap396 := d151
			snap397 := d152
			snap398 := d153
			snap399 := d154
			snap400 := d155
			snap401 := d156
			snap402 := d157
			snap403 := d158
			snap404 := d159
			snap405 := d160
			snap406 := d161
			snap407 := d162
			snap408 := d163
			snap409 := d164
			snap410 := d165
			snap411 := d166
			snap412 := d167
			snap413 := d168
			snap414 := d169
			snap415 := d170
			snap416 := d171
			snap417 := d172
			snap418 := d173
			snap419 := d174
			snap420 := d175
			snap421 := d176
			snap422 := d177
			snap423 := d178
			snap424 := d179
			snap425 := d180
			snap426 := d181
			snap427 := d182
			snap428 := d183
			snap429 := d184
			snap430 := d185
			snap431 := d186
			snap432 := d187
			snap433 := d188
			snap434 := d189
			snap435 := d190
			snap436 := d191
			snap437 := d192
			snap438 := d193
			snap439 := d194
			snap440 := d195
			snap441 := d196
			snap442 := d197
			snap443 := d198
			snap444 := d199
			snap445 := d200
			snap446 := d201
			snap447 := d202
			snap448 := d203
			snap449 := d204
			snap450 := d205
			snap451 := d206
			snap452 := d207
			snap453 := d208
			snap454 := d209
			snap455 := d210
			snap456 := d211
			snap457 := d212
			snap458 := d213
			snap459 := d214
			snap460 := d215
			snap461 := d216
			snap462 := d217
			snap463 := d218
			snap464 := d219
			snap465 := d220
			snap466 := d221
			snap467 := d222
			snap468 := d223
			snap469 := d224
			snap470 := d225
			snap471 := d226
			snap472 := d227
			snap473 := d228
			snap474 := d229
			snap475 := d230
			snap476 := d231
			snap477 := d232
			snap478 := d233
			snap479 := d234
			snap480 := d235
			snap481 := d236
			snap482 := d237
			snap483 := d238
			snap484 := d239
			snap485 := d240
			alloc486 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps244)
			}
			ctx.RestoreAllocState(alloc486)
			d0 = snap245
			d1 = snap246
			d2 = snap247
			d3 = snap248
			d4 = snap249
			d5 = snap250
			d6 = snap251
			d7 = snap252
			d8 = snap253
			d9 = snap254
			d10 = snap255
			d11 = snap256
			d12 = snap257
			d13 = snap258
			d14 = snap259
			d15 = snap260
			d16 = snap261
			d17 = snap262
			d18 = snap263
			d19 = snap264
			d20 = snap265
			d21 = snap266
			d22 = snap267
			d23 = snap268
			d24 = snap269
			d25 = snap270
			d26 = snap271
			d27 = snap272
			d28 = snap273
			d29 = snap274
			d30 = snap275
			d31 = snap276
			d32 = snap277
			d33 = snap278
			d34 = snap279
			d35 = snap280
			d36 = snap281
			d37 = snap282
			d38 = snap283
			d39 = snap284
			d40 = snap285
			d41 = snap286
			d42 = snap287
			d43 = snap288
			d44 = snap289
			d45 = snap290
			d46 = snap291
			d47 = snap292
			d48 = snap293
			d49 = snap294
			d50 = snap295
			d51 = snap296
			d52 = snap297
			d53 = snap298
			d54 = snap299
			d55 = snap300
			d56 = snap301
			d57 = snap302
			d58 = snap303
			d59 = snap304
			d60 = snap305
			d61 = snap306
			d62 = snap307
			d63 = snap308
			d64 = snap309
			d65 = snap310
			d66 = snap311
			d67 = snap312
			d68 = snap313
			d69 = snap314
			d70 = snap315
			d71 = snap316
			d72 = snap317
			d73 = snap318
			d74 = snap319
			d75 = snap320
			d76 = snap321
			d77 = snap322
			d78 = snap323
			d79 = snap324
			d80 = snap325
			d81 = snap326
			d82 = snap327
			d83 = snap328
			d84 = snap329
			d85 = snap330
			d86 = snap331
			d87 = snap332
			d88 = snap333
			d89 = snap334
			d90 = snap335
			d91 = snap336
			d92 = snap337
			d93 = snap338
			d94 = snap339
			d95 = snap340
			d96 = snap341
			d97 = snap342
			d98 = snap343
			d99 = snap344
			d100 = snap345
			d101 = snap346
			d102 = snap347
			d103 = snap348
			d104 = snap349
			d105 = snap350
			d106 = snap351
			d107 = snap352
			d108 = snap353
			d109 = snap354
			d110 = snap355
			d111 = snap356
			d112 = snap357
			d113 = snap358
			d114 = snap359
			d115 = snap360
			d116 = snap361
			d117 = snap362
			d118 = snap363
			d119 = snap364
			d120 = snap365
			d121 = snap366
			d122 = snap367
			d123 = snap368
			d124 = snap369
			d125 = snap370
			d126 = snap371
			d127 = snap372
			d128 = snap373
			d129 = snap374
			d130 = snap375
			d131 = snap376
			d132 = snap377
			d133 = snap378
			d134 = snap379
			d135 = snap380
			d136 = snap381
			d137 = snap382
			d138 = snap383
			d139 = snap384
			d140 = snap385
			d141 = snap386
			d142 = snap387
			d143 = snap388
			d144 = snap389
			d145 = snap390
			d146 = snap391
			d147 = snap392
			d148 = snap393
			d149 = snap394
			d150 = snap395
			d151 = snap396
			d152 = snap397
			d153 = snap398
			d154 = snap399
			d155 = snap400
			d156 = snap401
			d157 = snap402
			d158 = snap403
			d159 = snap404
			d160 = snap405
			d161 = snap406
			d162 = snap407
			d163 = snap408
			d164 = snap409
			d165 = snap410
			d166 = snap411
			d167 = snap412
			d168 = snap413
			d169 = snap414
			d170 = snap415
			d171 = snap416
			d172 = snap417
			d173 = snap418
			d174 = snap419
			d175 = snap420
			d176 = snap421
			d177 = snap422
			d178 = snap423
			d179 = snap424
			d180 = snap425
			d181 = snap426
			d182 = snap427
			d183 = snap428
			d184 = snap429
			d185 = snap430
			d186 = snap431
			d187 = snap432
			d188 = snap433
			d189 = snap434
			d190 = snap435
			d191 = snap436
			d192 = snap437
			d193 = snap438
			d194 = snap439
			d195 = snap440
			d196 = snap441
			d197 = snap442
			d198 = snap443
			d199 = snap444
			d200 = snap445
			d201 = snap446
			d202 = snap447
			d203 = snap448
			d204 = snap449
			d205 = snap450
			d206 = snap451
			d207 = snap452
			d208 = snap453
			d209 = snap454
			d210 = snap455
			d211 = snap456
			d212 = snap457
			d213 = snap458
			d214 = snap459
			d215 = snap460
			d216 = snap461
			d217 = snap462
			d218 = snap463
			d219 = snap464
			d220 = snap465
			d221 = snap466
			d222 = snap467
			d223 = snap468
			d224 = snap469
			d225 = snap470
			d226 = snap471
			d227 = snap472
			d228 = snap473
			d229 = snap474
			d230 = snap475
			d231 = snap476
			d232 = snap477
			d233 = snap478
			d234 = snap479
			d235 = snap480
			d236 = snap481
			d237 = snap482
			d238 = snap483
			d239 = snap484
			d240 = snap485
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps243)
			}
			return result
			ctx.FreeDesc(&d238)
			return result
			}
			bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
			}
			bbs[1].VisitCount++
			if ps.General {
				if bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
				d47 = ps.OverlayValues[47]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
				d108 = ps.OverlayValues[108]
			}
			if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
				d109 = ps.OverlayValues[109]
			}
			if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
				d110 = ps.OverlayValues[110]
			}
			if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
				d111 = ps.OverlayValues[111]
			}
			if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
				d112 = ps.OverlayValues[112]
			}
			if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
				d113 = ps.OverlayValues[113]
			}
			if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
				d114 = ps.OverlayValues[114]
			}
			if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
				d115 = ps.OverlayValues[115]
			}
			if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
				d116 = ps.OverlayValues[116]
			}
			if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
				d117 = ps.OverlayValues[117]
			}
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != scm.LocNone {
				d118 = ps.OverlayValues[118]
			}
			if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
				d119 = ps.OverlayValues[119]
			}
			if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != scm.LocNone {
				d120 = ps.OverlayValues[120]
			}
			if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != scm.LocNone {
				d121 = ps.OverlayValues[121]
			}
			if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != scm.LocNone {
				d122 = ps.OverlayValues[122]
			}
			if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != scm.LocNone {
				d123 = ps.OverlayValues[123]
			}
			if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != scm.LocNone {
				d124 = ps.OverlayValues[124]
			}
			if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
				d125 = ps.OverlayValues[125]
			}
			if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
				d126 = ps.OverlayValues[126]
			}
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
				d128 = ps.OverlayValues[128]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
				d129 = ps.OverlayValues[129]
			}
			if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
				d130 = ps.OverlayValues[130]
			}
			if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
				d131 = ps.OverlayValues[131]
			}
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
				d133 = ps.OverlayValues[133]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
				d136 = ps.OverlayValues[136]
			}
			if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
				d137 = ps.OverlayValues[137]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
				d138 = ps.OverlayValues[138]
			}
			if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
				d139 = ps.OverlayValues[139]
			}
			if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
				d140 = ps.OverlayValues[140]
			}
			if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
				d141 = ps.OverlayValues[141]
			}
			if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
				d142 = ps.OverlayValues[142]
			}
			if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
				d143 = ps.OverlayValues[143]
			}
			if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
				d144 = ps.OverlayValues[144]
			}
			if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
				d145 = ps.OverlayValues[145]
			}
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
				d146 = ps.OverlayValues[146]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
				d155 = ps.OverlayValues[155]
			}
			if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
				d156 = ps.OverlayValues[156]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
				d159 = ps.OverlayValues[159]
			}
			if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
				d160 = ps.OverlayValues[160]
			}
			if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
				d161 = ps.OverlayValues[161]
			}
			if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
				d162 = ps.OverlayValues[162]
			}
			if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
				d163 = ps.OverlayValues[163]
			}
			if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
				d164 = ps.OverlayValues[164]
			}
			if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
				d165 = ps.OverlayValues[165]
			}
			if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
				d166 = ps.OverlayValues[166]
			}
			if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
				d167 = ps.OverlayValues[167]
			}
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
				d169 = ps.OverlayValues[169]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != scm.LocNone {
				d172 = ps.OverlayValues[172]
			}
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
				d176 = ps.OverlayValues[176]
			}
			if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
				d177 = ps.OverlayValues[177]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
				d179 = ps.OverlayValues[179]
			}
			if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
				d180 = ps.OverlayValues[180]
			}
			if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
				d181 = ps.OverlayValues[181]
			}
			if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
				d182 = ps.OverlayValues[182]
			}
			if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
				d183 = ps.OverlayValues[183]
			}
			if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
				d184 = ps.OverlayValues[184]
			}
			if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
				d185 = ps.OverlayValues[185]
			}
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
				d191 = ps.OverlayValues[191]
			}
			if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
				d192 = ps.OverlayValues[192]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
				d194 = ps.OverlayValues[194]
			}
			if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
				d195 = ps.OverlayValues[195]
			}
			if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
				d196 = ps.OverlayValues[196]
			}
			if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
				d197 = ps.OverlayValues[197]
			}
			if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
				d198 = ps.OverlayValues[198]
			}
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
				d205 = ps.OverlayValues[205]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != scm.LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != scm.LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != scm.LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != scm.LocNone {
				d211 = ps.OverlayValues[211]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != scm.LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != scm.LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != scm.LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != scm.LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != scm.LocNone {
				d216 = ps.OverlayValues[216]
			}
			if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != scm.LocNone {
				d217 = ps.OverlayValues[217]
			}
			if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != scm.LocNone {
				d218 = ps.OverlayValues[218]
			}
			if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != scm.LocNone {
				d219 = ps.OverlayValues[219]
			}
			if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
				d220 = ps.OverlayValues[220]
			}
			if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != scm.LocNone {
				d221 = ps.OverlayValues[221]
			}
			if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != scm.LocNone {
				d222 = ps.OverlayValues[222]
			}
			if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
				d223 = ps.OverlayValues[223]
			}
			if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != scm.LocNone {
				d224 = ps.OverlayValues[224]
			}
			if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
				d225 = ps.OverlayValues[225]
			}
			if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
				d226 = ps.OverlayValues[226]
			}
			if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
				d227 = ps.OverlayValues[227]
			}
			if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
				d228 = ps.OverlayValues[228]
			}
			if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
				d229 = ps.OverlayValues[229]
			}
			if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
				d230 = ps.OverlayValues[230]
			}
			if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
				d231 = ps.OverlayValues[231]
			}
			if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
				d232 = ps.OverlayValues[232]
			}
			if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
				d233 = ps.OverlayValues[233]
			}
			if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
				d234 = ps.OverlayValues[234]
			}
			if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
				d235 = ps.OverlayValues[235]
			}
			if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
				d236 = ps.OverlayValues[236]
			}
			if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
				d237 = ps.OverlayValues[237]
			}
			if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
				d238 = ps.OverlayValues[238]
			}
			if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
				d239 = ps.OverlayValues[239]
			}
			if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
				d240 = ps.OverlayValues[240]
			}
			ctx.ReclaimUntrackedRegs()
			d487 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d487)
			ctx.BindReg(r1, &d487)
			ctx.EmitMakeNil(d487)
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
					ps.General = true
					return bbs[2].RenderPS(ps)
				}
			}
			bbs[2].VisitCount++
			if ps.General {
				if bbs[2].Rendered {
					ctx.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
				d47 = ps.OverlayValues[47]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
				d108 = ps.OverlayValues[108]
			}
			if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
				d109 = ps.OverlayValues[109]
			}
			if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
				d110 = ps.OverlayValues[110]
			}
			if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
				d111 = ps.OverlayValues[111]
			}
			if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
				d112 = ps.OverlayValues[112]
			}
			if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
				d113 = ps.OverlayValues[113]
			}
			if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
				d114 = ps.OverlayValues[114]
			}
			if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
				d115 = ps.OverlayValues[115]
			}
			if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
				d116 = ps.OverlayValues[116]
			}
			if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
				d117 = ps.OverlayValues[117]
			}
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != scm.LocNone {
				d118 = ps.OverlayValues[118]
			}
			if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
				d119 = ps.OverlayValues[119]
			}
			if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != scm.LocNone {
				d120 = ps.OverlayValues[120]
			}
			if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != scm.LocNone {
				d121 = ps.OverlayValues[121]
			}
			if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != scm.LocNone {
				d122 = ps.OverlayValues[122]
			}
			if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != scm.LocNone {
				d123 = ps.OverlayValues[123]
			}
			if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != scm.LocNone {
				d124 = ps.OverlayValues[124]
			}
			if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
				d125 = ps.OverlayValues[125]
			}
			if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
				d126 = ps.OverlayValues[126]
			}
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
				d128 = ps.OverlayValues[128]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
				d129 = ps.OverlayValues[129]
			}
			if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
				d130 = ps.OverlayValues[130]
			}
			if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
				d131 = ps.OverlayValues[131]
			}
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
				d133 = ps.OverlayValues[133]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
				d136 = ps.OverlayValues[136]
			}
			if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
				d137 = ps.OverlayValues[137]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
				d138 = ps.OverlayValues[138]
			}
			if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
				d139 = ps.OverlayValues[139]
			}
			if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
				d140 = ps.OverlayValues[140]
			}
			if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
				d141 = ps.OverlayValues[141]
			}
			if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
				d142 = ps.OverlayValues[142]
			}
			if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
				d143 = ps.OverlayValues[143]
			}
			if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
				d144 = ps.OverlayValues[144]
			}
			if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
				d145 = ps.OverlayValues[145]
			}
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
				d146 = ps.OverlayValues[146]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
				d155 = ps.OverlayValues[155]
			}
			if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
				d156 = ps.OverlayValues[156]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
				d159 = ps.OverlayValues[159]
			}
			if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
				d160 = ps.OverlayValues[160]
			}
			if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
				d161 = ps.OverlayValues[161]
			}
			if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
				d162 = ps.OverlayValues[162]
			}
			if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
				d163 = ps.OverlayValues[163]
			}
			if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
				d164 = ps.OverlayValues[164]
			}
			if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
				d165 = ps.OverlayValues[165]
			}
			if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
				d166 = ps.OverlayValues[166]
			}
			if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
				d167 = ps.OverlayValues[167]
			}
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
				d169 = ps.OverlayValues[169]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != scm.LocNone {
				d172 = ps.OverlayValues[172]
			}
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
				d176 = ps.OverlayValues[176]
			}
			if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
				d177 = ps.OverlayValues[177]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
				d179 = ps.OverlayValues[179]
			}
			if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
				d180 = ps.OverlayValues[180]
			}
			if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
				d181 = ps.OverlayValues[181]
			}
			if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
				d182 = ps.OverlayValues[182]
			}
			if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
				d183 = ps.OverlayValues[183]
			}
			if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
				d184 = ps.OverlayValues[184]
			}
			if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
				d185 = ps.OverlayValues[185]
			}
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
				d191 = ps.OverlayValues[191]
			}
			if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
				d192 = ps.OverlayValues[192]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
				d194 = ps.OverlayValues[194]
			}
			if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
				d195 = ps.OverlayValues[195]
			}
			if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
				d196 = ps.OverlayValues[196]
			}
			if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
				d197 = ps.OverlayValues[197]
			}
			if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
				d198 = ps.OverlayValues[198]
			}
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
				d205 = ps.OverlayValues[205]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != scm.LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != scm.LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != scm.LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != scm.LocNone {
				d211 = ps.OverlayValues[211]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != scm.LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != scm.LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != scm.LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != scm.LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != scm.LocNone {
				d216 = ps.OverlayValues[216]
			}
			if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != scm.LocNone {
				d217 = ps.OverlayValues[217]
			}
			if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != scm.LocNone {
				d218 = ps.OverlayValues[218]
			}
			if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != scm.LocNone {
				d219 = ps.OverlayValues[219]
			}
			if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
				d220 = ps.OverlayValues[220]
			}
			if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != scm.LocNone {
				d221 = ps.OverlayValues[221]
			}
			if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != scm.LocNone {
				d222 = ps.OverlayValues[222]
			}
			if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
				d223 = ps.OverlayValues[223]
			}
			if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != scm.LocNone {
				d224 = ps.OverlayValues[224]
			}
			if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
				d225 = ps.OverlayValues[225]
			}
			if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
				d226 = ps.OverlayValues[226]
			}
			if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
				d227 = ps.OverlayValues[227]
			}
			if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
				d228 = ps.OverlayValues[228]
			}
			if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
				d229 = ps.OverlayValues[229]
			}
			if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
				d230 = ps.OverlayValues[230]
			}
			if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
				d231 = ps.OverlayValues[231]
			}
			if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
				d232 = ps.OverlayValues[232]
			}
			if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
				d233 = ps.OverlayValues[233]
			}
			if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
				d234 = ps.OverlayValues[234]
			}
			if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
				d235 = ps.OverlayValues[235]
			}
			if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
				d236 = ps.OverlayValues[236]
			}
			if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
				d237 = ps.OverlayValues[237]
			}
			if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
				d238 = ps.OverlayValues[238]
			}
			if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
				d239 = ps.OverlayValues[239]
			}
			if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
				d240 = ps.OverlayValues[240]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			ctx.ReclaimUntrackedRegs()
			d489 = d237
			d489.ID = 0
			d488 = ctx.EmitTagEqualsBorrowed(&d489, scm.TagString, scm.JITValueDesc{Loc: scm.LocAny})
			d490 = d488
			ctx.EnsureDesc(&d490)
			if d490.Loc != scm.LocImm && d490.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d490.Loc == scm.LocImm {
				if d490.Imm.Bool() {
			ps491 := scm.PhiState{General: ps.General}
			ps491.OverlayValues = make([]scm.JITValueDesc, 491)
			ps491.OverlayValues[0] = d0
			ps491.OverlayValues[1] = d1
			ps491.OverlayValues[2] = d2
			ps491.OverlayValues[3] = d3
			ps491.OverlayValues[4] = d4
			ps491.OverlayValues[5] = d5
			ps491.OverlayValues[6] = d6
			ps491.OverlayValues[7] = d7
			ps491.OverlayValues[8] = d8
			ps491.OverlayValues[9] = d9
			ps491.OverlayValues[10] = d10
			ps491.OverlayValues[11] = d11
			ps491.OverlayValues[12] = d12
			ps491.OverlayValues[13] = d13
			ps491.OverlayValues[14] = d14
			ps491.OverlayValues[15] = d15
			ps491.OverlayValues[16] = d16
			ps491.OverlayValues[17] = d17
			ps491.OverlayValues[18] = d18
			ps491.OverlayValues[19] = d19
			ps491.OverlayValues[20] = d20
			ps491.OverlayValues[21] = d21
			ps491.OverlayValues[22] = d22
			ps491.OverlayValues[23] = d23
			ps491.OverlayValues[24] = d24
			ps491.OverlayValues[25] = d25
			ps491.OverlayValues[26] = d26
			ps491.OverlayValues[27] = d27
			ps491.OverlayValues[28] = d28
			ps491.OverlayValues[29] = d29
			ps491.OverlayValues[30] = d30
			ps491.OverlayValues[31] = d31
			ps491.OverlayValues[32] = d32
			ps491.OverlayValues[33] = d33
			ps491.OverlayValues[34] = d34
			ps491.OverlayValues[35] = d35
			ps491.OverlayValues[36] = d36
			ps491.OverlayValues[37] = d37
			ps491.OverlayValues[38] = d38
			ps491.OverlayValues[39] = d39
			ps491.OverlayValues[40] = d40
			ps491.OverlayValues[41] = d41
			ps491.OverlayValues[42] = d42
			ps491.OverlayValues[43] = d43
			ps491.OverlayValues[44] = d44
			ps491.OverlayValues[45] = d45
			ps491.OverlayValues[46] = d46
			ps491.OverlayValues[47] = d47
			ps491.OverlayValues[48] = d48
			ps491.OverlayValues[49] = d49
			ps491.OverlayValues[50] = d50
			ps491.OverlayValues[51] = d51
			ps491.OverlayValues[52] = d52
			ps491.OverlayValues[53] = d53
			ps491.OverlayValues[54] = d54
			ps491.OverlayValues[55] = d55
			ps491.OverlayValues[56] = d56
			ps491.OverlayValues[57] = d57
			ps491.OverlayValues[58] = d58
			ps491.OverlayValues[59] = d59
			ps491.OverlayValues[60] = d60
			ps491.OverlayValues[61] = d61
			ps491.OverlayValues[62] = d62
			ps491.OverlayValues[63] = d63
			ps491.OverlayValues[64] = d64
			ps491.OverlayValues[65] = d65
			ps491.OverlayValues[66] = d66
			ps491.OverlayValues[67] = d67
			ps491.OverlayValues[68] = d68
			ps491.OverlayValues[69] = d69
			ps491.OverlayValues[70] = d70
			ps491.OverlayValues[71] = d71
			ps491.OverlayValues[72] = d72
			ps491.OverlayValues[73] = d73
			ps491.OverlayValues[74] = d74
			ps491.OverlayValues[75] = d75
			ps491.OverlayValues[76] = d76
			ps491.OverlayValues[77] = d77
			ps491.OverlayValues[78] = d78
			ps491.OverlayValues[79] = d79
			ps491.OverlayValues[80] = d80
			ps491.OverlayValues[81] = d81
			ps491.OverlayValues[82] = d82
			ps491.OverlayValues[83] = d83
			ps491.OverlayValues[84] = d84
			ps491.OverlayValues[85] = d85
			ps491.OverlayValues[86] = d86
			ps491.OverlayValues[87] = d87
			ps491.OverlayValues[88] = d88
			ps491.OverlayValues[89] = d89
			ps491.OverlayValues[90] = d90
			ps491.OverlayValues[91] = d91
			ps491.OverlayValues[92] = d92
			ps491.OverlayValues[93] = d93
			ps491.OverlayValues[94] = d94
			ps491.OverlayValues[95] = d95
			ps491.OverlayValues[96] = d96
			ps491.OverlayValues[97] = d97
			ps491.OverlayValues[98] = d98
			ps491.OverlayValues[99] = d99
			ps491.OverlayValues[100] = d100
			ps491.OverlayValues[101] = d101
			ps491.OverlayValues[102] = d102
			ps491.OverlayValues[103] = d103
			ps491.OverlayValues[104] = d104
			ps491.OverlayValues[105] = d105
			ps491.OverlayValues[106] = d106
			ps491.OverlayValues[107] = d107
			ps491.OverlayValues[108] = d108
			ps491.OverlayValues[109] = d109
			ps491.OverlayValues[110] = d110
			ps491.OverlayValues[111] = d111
			ps491.OverlayValues[112] = d112
			ps491.OverlayValues[113] = d113
			ps491.OverlayValues[114] = d114
			ps491.OverlayValues[115] = d115
			ps491.OverlayValues[116] = d116
			ps491.OverlayValues[117] = d117
			ps491.OverlayValues[118] = d118
			ps491.OverlayValues[119] = d119
			ps491.OverlayValues[120] = d120
			ps491.OverlayValues[121] = d121
			ps491.OverlayValues[122] = d122
			ps491.OverlayValues[123] = d123
			ps491.OverlayValues[124] = d124
			ps491.OverlayValues[125] = d125
			ps491.OverlayValues[126] = d126
			ps491.OverlayValues[127] = d127
			ps491.OverlayValues[128] = d128
			ps491.OverlayValues[129] = d129
			ps491.OverlayValues[130] = d130
			ps491.OverlayValues[131] = d131
			ps491.OverlayValues[132] = d132
			ps491.OverlayValues[133] = d133
			ps491.OverlayValues[134] = d134
			ps491.OverlayValues[135] = d135
			ps491.OverlayValues[136] = d136
			ps491.OverlayValues[137] = d137
			ps491.OverlayValues[138] = d138
			ps491.OverlayValues[139] = d139
			ps491.OverlayValues[140] = d140
			ps491.OverlayValues[141] = d141
			ps491.OverlayValues[142] = d142
			ps491.OverlayValues[143] = d143
			ps491.OverlayValues[144] = d144
			ps491.OverlayValues[145] = d145
			ps491.OverlayValues[146] = d146
			ps491.OverlayValues[147] = d147
			ps491.OverlayValues[148] = d148
			ps491.OverlayValues[149] = d149
			ps491.OverlayValues[150] = d150
			ps491.OverlayValues[151] = d151
			ps491.OverlayValues[152] = d152
			ps491.OverlayValues[153] = d153
			ps491.OverlayValues[154] = d154
			ps491.OverlayValues[155] = d155
			ps491.OverlayValues[156] = d156
			ps491.OverlayValues[157] = d157
			ps491.OverlayValues[158] = d158
			ps491.OverlayValues[159] = d159
			ps491.OverlayValues[160] = d160
			ps491.OverlayValues[161] = d161
			ps491.OverlayValues[162] = d162
			ps491.OverlayValues[163] = d163
			ps491.OverlayValues[164] = d164
			ps491.OverlayValues[165] = d165
			ps491.OverlayValues[166] = d166
			ps491.OverlayValues[167] = d167
			ps491.OverlayValues[168] = d168
			ps491.OverlayValues[169] = d169
			ps491.OverlayValues[170] = d170
			ps491.OverlayValues[171] = d171
			ps491.OverlayValues[172] = d172
			ps491.OverlayValues[173] = d173
			ps491.OverlayValues[174] = d174
			ps491.OverlayValues[175] = d175
			ps491.OverlayValues[176] = d176
			ps491.OverlayValues[177] = d177
			ps491.OverlayValues[178] = d178
			ps491.OverlayValues[179] = d179
			ps491.OverlayValues[180] = d180
			ps491.OverlayValues[181] = d181
			ps491.OverlayValues[182] = d182
			ps491.OverlayValues[183] = d183
			ps491.OverlayValues[184] = d184
			ps491.OverlayValues[185] = d185
			ps491.OverlayValues[186] = d186
			ps491.OverlayValues[187] = d187
			ps491.OverlayValues[188] = d188
			ps491.OverlayValues[189] = d189
			ps491.OverlayValues[190] = d190
			ps491.OverlayValues[191] = d191
			ps491.OverlayValues[192] = d192
			ps491.OverlayValues[193] = d193
			ps491.OverlayValues[194] = d194
			ps491.OverlayValues[195] = d195
			ps491.OverlayValues[196] = d196
			ps491.OverlayValues[197] = d197
			ps491.OverlayValues[198] = d198
			ps491.OverlayValues[199] = d199
			ps491.OverlayValues[200] = d200
			ps491.OverlayValues[201] = d201
			ps491.OverlayValues[202] = d202
			ps491.OverlayValues[203] = d203
			ps491.OverlayValues[204] = d204
			ps491.OverlayValues[205] = d205
			ps491.OverlayValues[206] = d206
			ps491.OverlayValues[207] = d207
			ps491.OverlayValues[208] = d208
			ps491.OverlayValues[209] = d209
			ps491.OverlayValues[210] = d210
			ps491.OverlayValues[211] = d211
			ps491.OverlayValues[212] = d212
			ps491.OverlayValues[213] = d213
			ps491.OverlayValues[214] = d214
			ps491.OverlayValues[215] = d215
			ps491.OverlayValues[216] = d216
			ps491.OverlayValues[217] = d217
			ps491.OverlayValues[218] = d218
			ps491.OverlayValues[219] = d219
			ps491.OverlayValues[220] = d220
			ps491.OverlayValues[221] = d221
			ps491.OverlayValues[222] = d222
			ps491.OverlayValues[223] = d223
			ps491.OverlayValues[224] = d224
			ps491.OverlayValues[225] = d225
			ps491.OverlayValues[226] = d226
			ps491.OverlayValues[227] = d227
			ps491.OverlayValues[228] = d228
			ps491.OverlayValues[229] = d229
			ps491.OverlayValues[230] = d230
			ps491.OverlayValues[231] = d231
			ps491.OverlayValues[232] = d232
			ps491.OverlayValues[233] = d233
			ps491.OverlayValues[234] = d234
			ps491.OverlayValues[235] = d235
			ps491.OverlayValues[236] = d236
			ps491.OverlayValues[237] = d237
			ps491.OverlayValues[238] = d238
			ps491.OverlayValues[239] = d239
			ps491.OverlayValues[240] = d240
			ps491.OverlayValues[487] = d487
			ps491.OverlayValues[488] = d488
			ps491.OverlayValues[489] = d489
			ps491.OverlayValues[490] = d490
					return bbs[4].RenderPS(ps491)
				}
			ps492 := scm.PhiState{General: ps.General}
			ps492.OverlayValues = make([]scm.JITValueDesc, 491)
			ps492.OverlayValues[0] = d0
			ps492.OverlayValues[1] = d1
			ps492.OverlayValues[2] = d2
			ps492.OverlayValues[3] = d3
			ps492.OverlayValues[4] = d4
			ps492.OverlayValues[5] = d5
			ps492.OverlayValues[6] = d6
			ps492.OverlayValues[7] = d7
			ps492.OverlayValues[8] = d8
			ps492.OverlayValues[9] = d9
			ps492.OverlayValues[10] = d10
			ps492.OverlayValues[11] = d11
			ps492.OverlayValues[12] = d12
			ps492.OverlayValues[13] = d13
			ps492.OverlayValues[14] = d14
			ps492.OverlayValues[15] = d15
			ps492.OverlayValues[16] = d16
			ps492.OverlayValues[17] = d17
			ps492.OverlayValues[18] = d18
			ps492.OverlayValues[19] = d19
			ps492.OverlayValues[20] = d20
			ps492.OverlayValues[21] = d21
			ps492.OverlayValues[22] = d22
			ps492.OverlayValues[23] = d23
			ps492.OverlayValues[24] = d24
			ps492.OverlayValues[25] = d25
			ps492.OverlayValues[26] = d26
			ps492.OverlayValues[27] = d27
			ps492.OverlayValues[28] = d28
			ps492.OverlayValues[29] = d29
			ps492.OverlayValues[30] = d30
			ps492.OverlayValues[31] = d31
			ps492.OverlayValues[32] = d32
			ps492.OverlayValues[33] = d33
			ps492.OverlayValues[34] = d34
			ps492.OverlayValues[35] = d35
			ps492.OverlayValues[36] = d36
			ps492.OverlayValues[37] = d37
			ps492.OverlayValues[38] = d38
			ps492.OverlayValues[39] = d39
			ps492.OverlayValues[40] = d40
			ps492.OverlayValues[41] = d41
			ps492.OverlayValues[42] = d42
			ps492.OverlayValues[43] = d43
			ps492.OverlayValues[44] = d44
			ps492.OverlayValues[45] = d45
			ps492.OverlayValues[46] = d46
			ps492.OverlayValues[47] = d47
			ps492.OverlayValues[48] = d48
			ps492.OverlayValues[49] = d49
			ps492.OverlayValues[50] = d50
			ps492.OverlayValues[51] = d51
			ps492.OverlayValues[52] = d52
			ps492.OverlayValues[53] = d53
			ps492.OverlayValues[54] = d54
			ps492.OverlayValues[55] = d55
			ps492.OverlayValues[56] = d56
			ps492.OverlayValues[57] = d57
			ps492.OverlayValues[58] = d58
			ps492.OverlayValues[59] = d59
			ps492.OverlayValues[60] = d60
			ps492.OverlayValues[61] = d61
			ps492.OverlayValues[62] = d62
			ps492.OverlayValues[63] = d63
			ps492.OverlayValues[64] = d64
			ps492.OverlayValues[65] = d65
			ps492.OverlayValues[66] = d66
			ps492.OverlayValues[67] = d67
			ps492.OverlayValues[68] = d68
			ps492.OverlayValues[69] = d69
			ps492.OverlayValues[70] = d70
			ps492.OverlayValues[71] = d71
			ps492.OverlayValues[72] = d72
			ps492.OverlayValues[73] = d73
			ps492.OverlayValues[74] = d74
			ps492.OverlayValues[75] = d75
			ps492.OverlayValues[76] = d76
			ps492.OverlayValues[77] = d77
			ps492.OverlayValues[78] = d78
			ps492.OverlayValues[79] = d79
			ps492.OverlayValues[80] = d80
			ps492.OverlayValues[81] = d81
			ps492.OverlayValues[82] = d82
			ps492.OverlayValues[83] = d83
			ps492.OverlayValues[84] = d84
			ps492.OverlayValues[85] = d85
			ps492.OverlayValues[86] = d86
			ps492.OverlayValues[87] = d87
			ps492.OverlayValues[88] = d88
			ps492.OverlayValues[89] = d89
			ps492.OverlayValues[90] = d90
			ps492.OverlayValues[91] = d91
			ps492.OverlayValues[92] = d92
			ps492.OverlayValues[93] = d93
			ps492.OverlayValues[94] = d94
			ps492.OverlayValues[95] = d95
			ps492.OverlayValues[96] = d96
			ps492.OverlayValues[97] = d97
			ps492.OverlayValues[98] = d98
			ps492.OverlayValues[99] = d99
			ps492.OverlayValues[100] = d100
			ps492.OverlayValues[101] = d101
			ps492.OverlayValues[102] = d102
			ps492.OverlayValues[103] = d103
			ps492.OverlayValues[104] = d104
			ps492.OverlayValues[105] = d105
			ps492.OverlayValues[106] = d106
			ps492.OverlayValues[107] = d107
			ps492.OverlayValues[108] = d108
			ps492.OverlayValues[109] = d109
			ps492.OverlayValues[110] = d110
			ps492.OverlayValues[111] = d111
			ps492.OverlayValues[112] = d112
			ps492.OverlayValues[113] = d113
			ps492.OverlayValues[114] = d114
			ps492.OverlayValues[115] = d115
			ps492.OverlayValues[116] = d116
			ps492.OverlayValues[117] = d117
			ps492.OverlayValues[118] = d118
			ps492.OverlayValues[119] = d119
			ps492.OverlayValues[120] = d120
			ps492.OverlayValues[121] = d121
			ps492.OverlayValues[122] = d122
			ps492.OverlayValues[123] = d123
			ps492.OverlayValues[124] = d124
			ps492.OverlayValues[125] = d125
			ps492.OverlayValues[126] = d126
			ps492.OverlayValues[127] = d127
			ps492.OverlayValues[128] = d128
			ps492.OverlayValues[129] = d129
			ps492.OverlayValues[130] = d130
			ps492.OverlayValues[131] = d131
			ps492.OverlayValues[132] = d132
			ps492.OverlayValues[133] = d133
			ps492.OverlayValues[134] = d134
			ps492.OverlayValues[135] = d135
			ps492.OverlayValues[136] = d136
			ps492.OverlayValues[137] = d137
			ps492.OverlayValues[138] = d138
			ps492.OverlayValues[139] = d139
			ps492.OverlayValues[140] = d140
			ps492.OverlayValues[141] = d141
			ps492.OverlayValues[142] = d142
			ps492.OverlayValues[143] = d143
			ps492.OverlayValues[144] = d144
			ps492.OverlayValues[145] = d145
			ps492.OverlayValues[146] = d146
			ps492.OverlayValues[147] = d147
			ps492.OverlayValues[148] = d148
			ps492.OverlayValues[149] = d149
			ps492.OverlayValues[150] = d150
			ps492.OverlayValues[151] = d151
			ps492.OverlayValues[152] = d152
			ps492.OverlayValues[153] = d153
			ps492.OverlayValues[154] = d154
			ps492.OverlayValues[155] = d155
			ps492.OverlayValues[156] = d156
			ps492.OverlayValues[157] = d157
			ps492.OverlayValues[158] = d158
			ps492.OverlayValues[159] = d159
			ps492.OverlayValues[160] = d160
			ps492.OverlayValues[161] = d161
			ps492.OverlayValues[162] = d162
			ps492.OverlayValues[163] = d163
			ps492.OverlayValues[164] = d164
			ps492.OverlayValues[165] = d165
			ps492.OverlayValues[166] = d166
			ps492.OverlayValues[167] = d167
			ps492.OverlayValues[168] = d168
			ps492.OverlayValues[169] = d169
			ps492.OverlayValues[170] = d170
			ps492.OverlayValues[171] = d171
			ps492.OverlayValues[172] = d172
			ps492.OverlayValues[173] = d173
			ps492.OverlayValues[174] = d174
			ps492.OverlayValues[175] = d175
			ps492.OverlayValues[176] = d176
			ps492.OverlayValues[177] = d177
			ps492.OverlayValues[178] = d178
			ps492.OverlayValues[179] = d179
			ps492.OverlayValues[180] = d180
			ps492.OverlayValues[181] = d181
			ps492.OverlayValues[182] = d182
			ps492.OverlayValues[183] = d183
			ps492.OverlayValues[184] = d184
			ps492.OverlayValues[185] = d185
			ps492.OverlayValues[186] = d186
			ps492.OverlayValues[187] = d187
			ps492.OverlayValues[188] = d188
			ps492.OverlayValues[189] = d189
			ps492.OverlayValues[190] = d190
			ps492.OverlayValues[191] = d191
			ps492.OverlayValues[192] = d192
			ps492.OverlayValues[193] = d193
			ps492.OverlayValues[194] = d194
			ps492.OverlayValues[195] = d195
			ps492.OverlayValues[196] = d196
			ps492.OverlayValues[197] = d197
			ps492.OverlayValues[198] = d198
			ps492.OverlayValues[199] = d199
			ps492.OverlayValues[200] = d200
			ps492.OverlayValues[201] = d201
			ps492.OverlayValues[202] = d202
			ps492.OverlayValues[203] = d203
			ps492.OverlayValues[204] = d204
			ps492.OverlayValues[205] = d205
			ps492.OverlayValues[206] = d206
			ps492.OverlayValues[207] = d207
			ps492.OverlayValues[208] = d208
			ps492.OverlayValues[209] = d209
			ps492.OverlayValues[210] = d210
			ps492.OverlayValues[211] = d211
			ps492.OverlayValues[212] = d212
			ps492.OverlayValues[213] = d213
			ps492.OverlayValues[214] = d214
			ps492.OverlayValues[215] = d215
			ps492.OverlayValues[216] = d216
			ps492.OverlayValues[217] = d217
			ps492.OverlayValues[218] = d218
			ps492.OverlayValues[219] = d219
			ps492.OverlayValues[220] = d220
			ps492.OverlayValues[221] = d221
			ps492.OverlayValues[222] = d222
			ps492.OverlayValues[223] = d223
			ps492.OverlayValues[224] = d224
			ps492.OverlayValues[225] = d225
			ps492.OverlayValues[226] = d226
			ps492.OverlayValues[227] = d227
			ps492.OverlayValues[228] = d228
			ps492.OverlayValues[229] = d229
			ps492.OverlayValues[230] = d230
			ps492.OverlayValues[231] = d231
			ps492.OverlayValues[232] = d232
			ps492.OverlayValues[233] = d233
			ps492.OverlayValues[234] = d234
			ps492.OverlayValues[235] = d235
			ps492.OverlayValues[236] = d236
			ps492.OverlayValues[237] = d237
			ps492.OverlayValues[238] = d238
			ps492.OverlayValues[239] = d239
			ps492.OverlayValues[240] = d240
			ps492.OverlayValues[487] = d487
			ps492.OverlayValues[488] = d488
			ps492.OverlayValues[489] = d489
			ps492.OverlayValues[490] = d490
				return bbs[3].RenderPS(ps492)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl70 := ctx.ReserveLabel()
			lbl71 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d490.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl70)
			ctx.EmitJmp(lbl71)
			ctx.MarkLabel(lbl70)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl71)
			ctx.EmitJmp(lbl4)
			ps493 := scm.PhiState{General: true}
			ps493.OverlayValues = make([]scm.JITValueDesc, 491)
			ps493.OverlayValues[0] = d0
			ps493.OverlayValues[1] = d1
			ps493.OverlayValues[2] = d2
			ps493.OverlayValues[3] = d3
			ps493.OverlayValues[4] = d4
			ps493.OverlayValues[5] = d5
			ps493.OverlayValues[6] = d6
			ps493.OverlayValues[7] = d7
			ps493.OverlayValues[8] = d8
			ps493.OverlayValues[9] = d9
			ps493.OverlayValues[10] = d10
			ps493.OverlayValues[11] = d11
			ps493.OverlayValues[12] = d12
			ps493.OverlayValues[13] = d13
			ps493.OverlayValues[14] = d14
			ps493.OverlayValues[15] = d15
			ps493.OverlayValues[16] = d16
			ps493.OverlayValues[17] = d17
			ps493.OverlayValues[18] = d18
			ps493.OverlayValues[19] = d19
			ps493.OverlayValues[20] = d20
			ps493.OverlayValues[21] = d21
			ps493.OverlayValues[22] = d22
			ps493.OverlayValues[23] = d23
			ps493.OverlayValues[24] = d24
			ps493.OverlayValues[25] = d25
			ps493.OverlayValues[26] = d26
			ps493.OverlayValues[27] = d27
			ps493.OverlayValues[28] = d28
			ps493.OverlayValues[29] = d29
			ps493.OverlayValues[30] = d30
			ps493.OverlayValues[31] = d31
			ps493.OverlayValues[32] = d32
			ps493.OverlayValues[33] = d33
			ps493.OverlayValues[34] = d34
			ps493.OverlayValues[35] = d35
			ps493.OverlayValues[36] = d36
			ps493.OverlayValues[37] = d37
			ps493.OverlayValues[38] = d38
			ps493.OverlayValues[39] = d39
			ps493.OverlayValues[40] = d40
			ps493.OverlayValues[41] = d41
			ps493.OverlayValues[42] = d42
			ps493.OverlayValues[43] = d43
			ps493.OverlayValues[44] = d44
			ps493.OverlayValues[45] = d45
			ps493.OverlayValues[46] = d46
			ps493.OverlayValues[47] = d47
			ps493.OverlayValues[48] = d48
			ps493.OverlayValues[49] = d49
			ps493.OverlayValues[50] = d50
			ps493.OverlayValues[51] = d51
			ps493.OverlayValues[52] = d52
			ps493.OverlayValues[53] = d53
			ps493.OverlayValues[54] = d54
			ps493.OverlayValues[55] = d55
			ps493.OverlayValues[56] = d56
			ps493.OverlayValues[57] = d57
			ps493.OverlayValues[58] = d58
			ps493.OverlayValues[59] = d59
			ps493.OverlayValues[60] = d60
			ps493.OverlayValues[61] = d61
			ps493.OverlayValues[62] = d62
			ps493.OverlayValues[63] = d63
			ps493.OverlayValues[64] = d64
			ps493.OverlayValues[65] = d65
			ps493.OverlayValues[66] = d66
			ps493.OverlayValues[67] = d67
			ps493.OverlayValues[68] = d68
			ps493.OverlayValues[69] = d69
			ps493.OverlayValues[70] = d70
			ps493.OverlayValues[71] = d71
			ps493.OverlayValues[72] = d72
			ps493.OverlayValues[73] = d73
			ps493.OverlayValues[74] = d74
			ps493.OverlayValues[75] = d75
			ps493.OverlayValues[76] = d76
			ps493.OverlayValues[77] = d77
			ps493.OverlayValues[78] = d78
			ps493.OverlayValues[79] = d79
			ps493.OverlayValues[80] = d80
			ps493.OverlayValues[81] = d81
			ps493.OverlayValues[82] = d82
			ps493.OverlayValues[83] = d83
			ps493.OverlayValues[84] = d84
			ps493.OverlayValues[85] = d85
			ps493.OverlayValues[86] = d86
			ps493.OverlayValues[87] = d87
			ps493.OverlayValues[88] = d88
			ps493.OverlayValues[89] = d89
			ps493.OverlayValues[90] = d90
			ps493.OverlayValues[91] = d91
			ps493.OverlayValues[92] = d92
			ps493.OverlayValues[93] = d93
			ps493.OverlayValues[94] = d94
			ps493.OverlayValues[95] = d95
			ps493.OverlayValues[96] = d96
			ps493.OverlayValues[97] = d97
			ps493.OverlayValues[98] = d98
			ps493.OverlayValues[99] = d99
			ps493.OverlayValues[100] = d100
			ps493.OverlayValues[101] = d101
			ps493.OverlayValues[102] = d102
			ps493.OverlayValues[103] = d103
			ps493.OverlayValues[104] = d104
			ps493.OverlayValues[105] = d105
			ps493.OverlayValues[106] = d106
			ps493.OverlayValues[107] = d107
			ps493.OverlayValues[108] = d108
			ps493.OverlayValues[109] = d109
			ps493.OverlayValues[110] = d110
			ps493.OverlayValues[111] = d111
			ps493.OverlayValues[112] = d112
			ps493.OverlayValues[113] = d113
			ps493.OverlayValues[114] = d114
			ps493.OverlayValues[115] = d115
			ps493.OverlayValues[116] = d116
			ps493.OverlayValues[117] = d117
			ps493.OverlayValues[118] = d118
			ps493.OverlayValues[119] = d119
			ps493.OverlayValues[120] = d120
			ps493.OverlayValues[121] = d121
			ps493.OverlayValues[122] = d122
			ps493.OverlayValues[123] = d123
			ps493.OverlayValues[124] = d124
			ps493.OverlayValues[125] = d125
			ps493.OverlayValues[126] = d126
			ps493.OverlayValues[127] = d127
			ps493.OverlayValues[128] = d128
			ps493.OverlayValues[129] = d129
			ps493.OverlayValues[130] = d130
			ps493.OverlayValues[131] = d131
			ps493.OverlayValues[132] = d132
			ps493.OverlayValues[133] = d133
			ps493.OverlayValues[134] = d134
			ps493.OverlayValues[135] = d135
			ps493.OverlayValues[136] = d136
			ps493.OverlayValues[137] = d137
			ps493.OverlayValues[138] = d138
			ps493.OverlayValues[139] = d139
			ps493.OverlayValues[140] = d140
			ps493.OverlayValues[141] = d141
			ps493.OverlayValues[142] = d142
			ps493.OverlayValues[143] = d143
			ps493.OverlayValues[144] = d144
			ps493.OverlayValues[145] = d145
			ps493.OverlayValues[146] = d146
			ps493.OverlayValues[147] = d147
			ps493.OverlayValues[148] = d148
			ps493.OverlayValues[149] = d149
			ps493.OverlayValues[150] = d150
			ps493.OverlayValues[151] = d151
			ps493.OverlayValues[152] = d152
			ps493.OverlayValues[153] = d153
			ps493.OverlayValues[154] = d154
			ps493.OverlayValues[155] = d155
			ps493.OverlayValues[156] = d156
			ps493.OverlayValues[157] = d157
			ps493.OverlayValues[158] = d158
			ps493.OverlayValues[159] = d159
			ps493.OverlayValues[160] = d160
			ps493.OverlayValues[161] = d161
			ps493.OverlayValues[162] = d162
			ps493.OverlayValues[163] = d163
			ps493.OverlayValues[164] = d164
			ps493.OverlayValues[165] = d165
			ps493.OverlayValues[166] = d166
			ps493.OverlayValues[167] = d167
			ps493.OverlayValues[168] = d168
			ps493.OverlayValues[169] = d169
			ps493.OverlayValues[170] = d170
			ps493.OverlayValues[171] = d171
			ps493.OverlayValues[172] = d172
			ps493.OverlayValues[173] = d173
			ps493.OverlayValues[174] = d174
			ps493.OverlayValues[175] = d175
			ps493.OverlayValues[176] = d176
			ps493.OverlayValues[177] = d177
			ps493.OverlayValues[178] = d178
			ps493.OverlayValues[179] = d179
			ps493.OverlayValues[180] = d180
			ps493.OverlayValues[181] = d181
			ps493.OverlayValues[182] = d182
			ps493.OverlayValues[183] = d183
			ps493.OverlayValues[184] = d184
			ps493.OverlayValues[185] = d185
			ps493.OverlayValues[186] = d186
			ps493.OverlayValues[187] = d187
			ps493.OverlayValues[188] = d188
			ps493.OverlayValues[189] = d189
			ps493.OverlayValues[190] = d190
			ps493.OverlayValues[191] = d191
			ps493.OverlayValues[192] = d192
			ps493.OverlayValues[193] = d193
			ps493.OverlayValues[194] = d194
			ps493.OverlayValues[195] = d195
			ps493.OverlayValues[196] = d196
			ps493.OverlayValues[197] = d197
			ps493.OverlayValues[198] = d198
			ps493.OverlayValues[199] = d199
			ps493.OverlayValues[200] = d200
			ps493.OverlayValues[201] = d201
			ps493.OverlayValues[202] = d202
			ps493.OverlayValues[203] = d203
			ps493.OverlayValues[204] = d204
			ps493.OverlayValues[205] = d205
			ps493.OverlayValues[206] = d206
			ps493.OverlayValues[207] = d207
			ps493.OverlayValues[208] = d208
			ps493.OverlayValues[209] = d209
			ps493.OverlayValues[210] = d210
			ps493.OverlayValues[211] = d211
			ps493.OverlayValues[212] = d212
			ps493.OverlayValues[213] = d213
			ps493.OverlayValues[214] = d214
			ps493.OverlayValues[215] = d215
			ps493.OverlayValues[216] = d216
			ps493.OverlayValues[217] = d217
			ps493.OverlayValues[218] = d218
			ps493.OverlayValues[219] = d219
			ps493.OverlayValues[220] = d220
			ps493.OverlayValues[221] = d221
			ps493.OverlayValues[222] = d222
			ps493.OverlayValues[223] = d223
			ps493.OverlayValues[224] = d224
			ps493.OverlayValues[225] = d225
			ps493.OverlayValues[226] = d226
			ps493.OverlayValues[227] = d227
			ps493.OverlayValues[228] = d228
			ps493.OverlayValues[229] = d229
			ps493.OverlayValues[230] = d230
			ps493.OverlayValues[231] = d231
			ps493.OverlayValues[232] = d232
			ps493.OverlayValues[233] = d233
			ps493.OverlayValues[234] = d234
			ps493.OverlayValues[235] = d235
			ps493.OverlayValues[236] = d236
			ps493.OverlayValues[237] = d237
			ps493.OverlayValues[238] = d238
			ps493.OverlayValues[239] = d239
			ps493.OverlayValues[240] = d240
			ps493.OverlayValues[487] = d487
			ps493.OverlayValues[488] = d488
			ps493.OverlayValues[489] = d489
			ps493.OverlayValues[490] = d490
			ps494 := scm.PhiState{General: true}
			ps494.OverlayValues = make([]scm.JITValueDesc, 491)
			ps494.OverlayValues[0] = d0
			ps494.OverlayValues[1] = d1
			ps494.OverlayValues[2] = d2
			ps494.OverlayValues[3] = d3
			ps494.OverlayValues[4] = d4
			ps494.OverlayValues[5] = d5
			ps494.OverlayValues[6] = d6
			ps494.OverlayValues[7] = d7
			ps494.OverlayValues[8] = d8
			ps494.OverlayValues[9] = d9
			ps494.OverlayValues[10] = d10
			ps494.OverlayValues[11] = d11
			ps494.OverlayValues[12] = d12
			ps494.OverlayValues[13] = d13
			ps494.OverlayValues[14] = d14
			ps494.OverlayValues[15] = d15
			ps494.OverlayValues[16] = d16
			ps494.OverlayValues[17] = d17
			ps494.OverlayValues[18] = d18
			ps494.OverlayValues[19] = d19
			ps494.OverlayValues[20] = d20
			ps494.OverlayValues[21] = d21
			ps494.OverlayValues[22] = d22
			ps494.OverlayValues[23] = d23
			ps494.OverlayValues[24] = d24
			ps494.OverlayValues[25] = d25
			ps494.OverlayValues[26] = d26
			ps494.OverlayValues[27] = d27
			ps494.OverlayValues[28] = d28
			ps494.OverlayValues[29] = d29
			ps494.OverlayValues[30] = d30
			ps494.OverlayValues[31] = d31
			ps494.OverlayValues[32] = d32
			ps494.OverlayValues[33] = d33
			ps494.OverlayValues[34] = d34
			ps494.OverlayValues[35] = d35
			ps494.OverlayValues[36] = d36
			ps494.OverlayValues[37] = d37
			ps494.OverlayValues[38] = d38
			ps494.OverlayValues[39] = d39
			ps494.OverlayValues[40] = d40
			ps494.OverlayValues[41] = d41
			ps494.OverlayValues[42] = d42
			ps494.OverlayValues[43] = d43
			ps494.OverlayValues[44] = d44
			ps494.OverlayValues[45] = d45
			ps494.OverlayValues[46] = d46
			ps494.OverlayValues[47] = d47
			ps494.OverlayValues[48] = d48
			ps494.OverlayValues[49] = d49
			ps494.OverlayValues[50] = d50
			ps494.OverlayValues[51] = d51
			ps494.OverlayValues[52] = d52
			ps494.OverlayValues[53] = d53
			ps494.OverlayValues[54] = d54
			ps494.OverlayValues[55] = d55
			ps494.OverlayValues[56] = d56
			ps494.OverlayValues[57] = d57
			ps494.OverlayValues[58] = d58
			ps494.OverlayValues[59] = d59
			ps494.OverlayValues[60] = d60
			ps494.OverlayValues[61] = d61
			ps494.OverlayValues[62] = d62
			ps494.OverlayValues[63] = d63
			ps494.OverlayValues[64] = d64
			ps494.OverlayValues[65] = d65
			ps494.OverlayValues[66] = d66
			ps494.OverlayValues[67] = d67
			ps494.OverlayValues[68] = d68
			ps494.OverlayValues[69] = d69
			ps494.OverlayValues[70] = d70
			ps494.OverlayValues[71] = d71
			ps494.OverlayValues[72] = d72
			ps494.OverlayValues[73] = d73
			ps494.OverlayValues[74] = d74
			ps494.OverlayValues[75] = d75
			ps494.OverlayValues[76] = d76
			ps494.OverlayValues[77] = d77
			ps494.OverlayValues[78] = d78
			ps494.OverlayValues[79] = d79
			ps494.OverlayValues[80] = d80
			ps494.OverlayValues[81] = d81
			ps494.OverlayValues[82] = d82
			ps494.OverlayValues[83] = d83
			ps494.OverlayValues[84] = d84
			ps494.OverlayValues[85] = d85
			ps494.OverlayValues[86] = d86
			ps494.OverlayValues[87] = d87
			ps494.OverlayValues[88] = d88
			ps494.OverlayValues[89] = d89
			ps494.OverlayValues[90] = d90
			ps494.OverlayValues[91] = d91
			ps494.OverlayValues[92] = d92
			ps494.OverlayValues[93] = d93
			ps494.OverlayValues[94] = d94
			ps494.OverlayValues[95] = d95
			ps494.OverlayValues[96] = d96
			ps494.OverlayValues[97] = d97
			ps494.OverlayValues[98] = d98
			ps494.OverlayValues[99] = d99
			ps494.OverlayValues[100] = d100
			ps494.OverlayValues[101] = d101
			ps494.OverlayValues[102] = d102
			ps494.OverlayValues[103] = d103
			ps494.OverlayValues[104] = d104
			ps494.OverlayValues[105] = d105
			ps494.OverlayValues[106] = d106
			ps494.OverlayValues[107] = d107
			ps494.OverlayValues[108] = d108
			ps494.OverlayValues[109] = d109
			ps494.OverlayValues[110] = d110
			ps494.OverlayValues[111] = d111
			ps494.OverlayValues[112] = d112
			ps494.OverlayValues[113] = d113
			ps494.OverlayValues[114] = d114
			ps494.OverlayValues[115] = d115
			ps494.OverlayValues[116] = d116
			ps494.OverlayValues[117] = d117
			ps494.OverlayValues[118] = d118
			ps494.OverlayValues[119] = d119
			ps494.OverlayValues[120] = d120
			ps494.OverlayValues[121] = d121
			ps494.OverlayValues[122] = d122
			ps494.OverlayValues[123] = d123
			ps494.OverlayValues[124] = d124
			ps494.OverlayValues[125] = d125
			ps494.OverlayValues[126] = d126
			ps494.OverlayValues[127] = d127
			ps494.OverlayValues[128] = d128
			ps494.OverlayValues[129] = d129
			ps494.OverlayValues[130] = d130
			ps494.OverlayValues[131] = d131
			ps494.OverlayValues[132] = d132
			ps494.OverlayValues[133] = d133
			ps494.OverlayValues[134] = d134
			ps494.OverlayValues[135] = d135
			ps494.OverlayValues[136] = d136
			ps494.OverlayValues[137] = d137
			ps494.OverlayValues[138] = d138
			ps494.OverlayValues[139] = d139
			ps494.OverlayValues[140] = d140
			ps494.OverlayValues[141] = d141
			ps494.OverlayValues[142] = d142
			ps494.OverlayValues[143] = d143
			ps494.OverlayValues[144] = d144
			ps494.OverlayValues[145] = d145
			ps494.OverlayValues[146] = d146
			ps494.OverlayValues[147] = d147
			ps494.OverlayValues[148] = d148
			ps494.OverlayValues[149] = d149
			ps494.OverlayValues[150] = d150
			ps494.OverlayValues[151] = d151
			ps494.OverlayValues[152] = d152
			ps494.OverlayValues[153] = d153
			ps494.OverlayValues[154] = d154
			ps494.OverlayValues[155] = d155
			ps494.OverlayValues[156] = d156
			ps494.OverlayValues[157] = d157
			ps494.OverlayValues[158] = d158
			ps494.OverlayValues[159] = d159
			ps494.OverlayValues[160] = d160
			ps494.OverlayValues[161] = d161
			ps494.OverlayValues[162] = d162
			ps494.OverlayValues[163] = d163
			ps494.OverlayValues[164] = d164
			ps494.OverlayValues[165] = d165
			ps494.OverlayValues[166] = d166
			ps494.OverlayValues[167] = d167
			ps494.OverlayValues[168] = d168
			ps494.OverlayValues[169] = d169
			ps494.OverlayValues[170] = d170
			ps494.OverlayValues[171] = d171
			ps494.OverlayValues[172] = d172
			ps494.OverlayValues[173] = d173
			ps494.OverlayValues[174] = d174
			ps494.OverlayValues[175] = d175
			ps494.OverlayValues[176] = d176
			ps494.OverlayValues[177] = d177
			ps494.OverlayValues[178] = d178
			ps494.OverlayValues[179] = d179
			ps494.OverlayValues[180] = d180
			ps494.OverlayValues[181] = d181
			ps494.OverlayValues[182] = d182
			ps494.OverlayValues[183] = d183
			ps494.OverlayValues[184] = d184
			ps494.OverlayValues[185] = d185
			ps494.OverlayValues[186] = d186
			ps494.OverlayValues[187] = d187
			ps494.OverlayValues[188] = d188
			ps494.OverlayValues[189] = d189
			ps494.OverlayValues[190] = d190
			ps494.OverlayValues[191] = d191
			ps494.OverlayValues[192] = d192
			ps494.OverlayValues[193] = d193
			ps494.OverlayValues[194] = d194
			ps494.OverlayValues[195] = d195
			ps494.OverlayValues[196] = d196
			ps494.OverlayValues[197] = d197
			ps494.OverlayValues[198] = d198
			ps494.OverlayValues[199] = d199
			ps494.OverlayValues[200] = d200
			ps494.OverlayValues[201] = d201
			ps494.OverlayValues[202] = d202
			ps494.OverlayValues[203] = d203
			ps494.OverlayValues[204] = d204
			ps494.OverlayValues[205] = d205
			ps494.OverlayValues[206] = d206
			ps494.OverlayValues[207] = d207
			ps494.OverlayValues[208] = d208
			ps494.OverlayValues[209] = d209
			ps494.OverlayValues[210] = d210
			ps494.OverlayValues[211] = d211
			ps494.OverlayValues[212] = d212
			ps494.OverlayValues[213] = d213
			ps494.OverlayValues[214] = d214
			ps494.OverlayValues[215] = d215
			ps494.OverlayValues[216] = d216
			ps494.OverlayValues[217] = d217
			ps494.OverlayValues[218] = d218
			ps494.OverlayValues[219] = d219
			ps494.OverlayValues[220] = d220
			ps494.OverlayValues[221] = d221
			ps494.OverlayValues[222] = d222
			ps494.OverlayValues[223] = d223
			ps494.OverlayValues[224] = d224
			ps494.OverlayValues[225] = d225
			ps494.OverlayValues[226] = d226
			ps494.OverlayValues[227] = d227
			ps494.OverlayValues[228] = d228
			ps494.OverlayValues[229] = d229
			ps494.OverlayValues[230] = d230
			ps494.OverlayValues[231] = d231
			ps494.OverlayValues[232] = d232
			ps494.OverlayValues[233] = d233
			ps494.OverlayValues[234] = d234
			ps494.OverlayValues[235] = d235
			ps494.OverlayValues[236] = d236
			ps494.OverlayValues[237] = d237
			ps494.OverlayValues[238] = d238
			ps494.OverlayValues[239] = d239
			ps494.OverlayValues[240] = d240
			ps494.OverlayValues[487] = d487
			ps494.OverlayValues[488] = d488
			ps494.OverlayValues[489] = d489
			ps494.OverlayValues[490] = d490
			snap495 := d0
			snap496 := d1
			snap497 := d2
			snap498 := d3
			snap499 := d4
			snap500 := d5
			snap501 := d6
			snap502 := d7
			snap503 := d8
			snap504 := d9
			snap505 := d10
			snap506 := d11
			snap507 := d12
			snap508 := d13
			snap509 := d14
			snap510 := d15
			snap511 := d16
			snap512 := d17
			snap513 := d18
			snap514 := d19
			snap515 := d20
			snap516 := d21
			snap517 := d22
			snap518 := d23
			snap519 := d24
			snap520 := d25
			snap521 := d26
			snap522 := d27
			snap523 := d28
			snap524 := d29
			snap525 := d30
			snap526 := d31
			snap527 := d32
			snap528 := d33
			snap529 := d34
			snap530 := d35
			snap531 := d36
			snap532 := d37
			snap533 := d38
			snap534 := d39
			snap535 := d40
			snap536 := d41
			snap537 := d42
			snap538 := d43
			snap539 := d44
			snap540 := d45
			snap541 := d46
			snap542 := d47
			snap543 := d48
			snap544 := d49
			snap545 := d50
			snap546 := d51
			snap547 := d52
			snap548 := d53
			snap549 := d54
			snap550 := d55
			snap551 := d56
			snap552 := d57
			snap553 := d58
			snap554 := d59
			snap555 := d60
			snap556 := d61
			snap557 := d62
			snap558 := d63
			snap559 := d64
			snap560 := d65
			snap561 := d66
			snap562 := d67
			snap563 := d68
			snap564 := d69
			snap565 := d70
			snap566 := d71
			snap567 := d72
			snap568 := d73
			snap569 := d74
			snap570 := d75
			snap571 := d76
			snap572 := d77
			snap573 := d78
			snap574 := d79
			snap575 := d80
			snap576 := d81
			snap577 := d82
			snap578 := d83
			snap579 := d84
			snap580 := d85
			snap581 := d86
			snap582 := d87
			snap583 := d88
			snap584 := d89
			snap585 := d90
			snap586 := d91
			snap587 := d92
			snap588 := d93
			snap589 := d94
			snap590 := d95
			snap591 := d96
			snap592 := d97
			snap593 := d98
			snap594 := d99
			snap595 := d100
			snap596 := d101
			snap597 := d102
			snap598 := d103
			snap599 := d104
			snap600 := d105
			snap601 := d106
			snap602 := d107
			snap603 := d108
			snap604 := d109
			snap605 := d110
			snap606 := d111
			snap607 := d112
			snap608 := d113
			snap609 := d114
			snap610 := d115
			snap611 := d116
			snap612 := d117
			snap613 := d118
			snap614 := d119
			snap615 := d120
			snap616 := d121
			snap617 := d122
			snap618 := d123
			snap619 := d124
			snap620 := d125
			snap621 := d126
			snap622 := d127
			snap623 := d128
			snap624 := d129
			snap625 := d130
			snap626 := d131
			snap627 := d132
			snap628 := d133
			snap629 := d134
			snap630 := d135
			snap631 := d136
			snap632 := d137
			snap633 := d138
			snap634 := d139
			snap635 := d140
			snap636 := d141
			snap637 := d142
			snap638 := d143
			snap639 := d144
			snap640 := d145
			snap641 := d146
			snap642 := d147
			snap643 := d148
			snap644 := d149
			snap645 := d150
			snap646 := d151
			snap647 := d152
			snap648 := d153
			snap649 := d154
			snap650 := d155
			snap651 := d156
			snap652 := d157
			snap653 := d158
			snap654 := d159
			snap655 := d160
			snap656 := d161
			snap657 := d162
			snap658 := d163
			snap659 := d164
			snap660 := d165
			snap661 := d166
			snap662 := d167
			snap663 := d168
			snap664 := d169
			snap665 := d170
			snap666 := d171
			snap667 := d172
			snap668 := d173
			snap669 := d174
			snap670 := d175
			snap671 := d176
			snap672 := d177
			snap673 := d178
			snap674 := d179
			snap675 := d180
			snap676 := d181
			snap677 := d182
			snap678 := d183
			snap679 := d184
			snap680 := d185
			snap681 := d186
			snap682 := d187
			snap683 := d188
			snap684 := d189
			snap685 := d190
			snap686 := d191
			snap687 := d192
			snap688 := d193
			snap689 := d194
			snap690 := d195
			snap691 := d196
			snap692 := d197
			snap693 := d198
			snap694 := d199
			snap695 := d200
			snap696 := d201
			snap697 := d202
			snap698 := d203
			snap699 := d204
			snap700 := d205
			snap701 := d206
			snap702 := d207
			snap703 := d208
			snap704 := d209
			snap705 := d210
			snap706 := d211
			snap707 := d212
			snap708 := d213
			snap709 := d214
			snap710 := d215
			snap711 := d216
			snap712 := d217
			snap713 := d218
			snap714 := d219
			snap715 := d220
			snap716 := d221
			snap717 := d222
			snap718 := d223
			snap719 := d224
			snap720 := d225
			snap721 := d226
			snap722 := d227
			snap723 := d228
			snap724 := d229
			snap725 := d230
			snap726 := d231
			snap727 := d232
			snap728 := d233
			snap729 := d234
			snap730 := d235
			snap731 := d236
			snap732 := d237
			snap733 := d238
			snap734 := d239
			snap735 := d240
			snap736 := d487
			snap737 := d488
			snap738 := d489
			snap739 := d490
			alloc740 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps494)
			}
			ctx.RestoreAllocState(alloc740)
			d0 = snap495
			d1 = snap496
			d2 = snap497
			d3 = snap498
			d4 = snap499
			d5 = snap500
			d6 = snap501
			d7 = snap502
			d8 = snap503
			d9 = snap504
			d10 = snap505
			d11 = snap506
			d12 = snap507
			d13 = snap508
			d14 = snap509
			d15 = snap510
			d16 = snap511
			d17 = snap512
			d18 = snap513
			d19 = snap514
			d20 = snap515
			d21 = snap516
			d22 = snap517
			d23 = snap518
			d24 = snap519
			d25 = snap520
			d26 = snap521
			d27 = snap522
			d28 = snap523
			d29 = snap524
			d30 = snap525
			d31 = snap526
			d32 = snap527
			d33 = snap528
			d34 = snap529
			d35 = snap530
			d36 = snap531
			d37 = snap532
			d38 = snap533
			d39 = snap534
			d40 = snap535
			d41 = snap536
			d42 = snap537
			d43 = snap538
			d44 = snap539
			d45 = snap540
			d46 = snap541
			d47 = snap542
			d48 = snap543
			d49 = snap544
			d50 = snap545
			d51 = snap546
			d52 = snap547
			d53 = snap548
			d54 = snap549
			d55 = snap550
			d56 = snap551
			d57 = snap552
			d58 = snap553
			d59 = snap554
			d60 = snap555
			d61 = snap556
			d62 = snap557
			d63 = snap558
			d64 = snap559
			d65 = snap560
			d66 = snap561
			d67 = snap562
			d68 = snap563
			d69 = snap564
			d70 = snap565
			d71 = snap566
			d72 = snap567
			d73 = snap568
			d74 = snap569
			d75 = snap570
			d76 = snap571
			d77 = snap572
			d78 = snap573
			d79 = snap574
			d80 = snap575
			d81 = snap576
			d82 = snap577
			d83 = snap578
			d84 = snap579
			d85 = snap580
			d86 = snap581
			d87 = snap582
			d88 = snap583
			d89 = snap584
			d90 = snap585
			d91 = snap586
			d92 = snap587
			d93 = snap588
			d94 = snap589
			d95 = snap590
			d96 = snap591
			d97 = snap592
			d98 = snap593
			d99 = snap594
			d100 = snap595
			d101 = snap596
			d102 = snap597
			d103 = snap598
			d104 = snap599
			d105 = snap600
			d106 = snap601
			d107 = snap602
			d108 = snap603
			d109 = snap604
			d110 = snap605
			d111 = snap606
			d112 = snap607
			d113 = snap608
			d114 = snap609
			d115 = snap610
			d116 = snap611
			d117 = snap612
			d118 = snap613
			d119 = snap614
			d120 = snap615
			d121 = snap616
			d122 = snap617
			d123 = snap618
			d124 = snap619
			d125 = snap620
			d126 = snap621
			d127 = snap622
			d128 = snap623
			d129 = snap624
			d130 = snap625
			d131 = snap626
			d132 = snap627
			d133 = snap628
			d134 = snap629
			d135 = snap630
			d136 = snap631
			d137 = snap632
			d138 = snap633
			d139 = snap634
			d140 = snap635
			d141 = snap636
			d142 = snap637
			d143 = snap638
			d144 = snap639
			d145 = snap640
			d146 = snap641
			d147 = snap642
			d148 = snap643
			d149 = snap644
			d150 = snap645
			d151 = snap646
			d152 = snap647
			d153 = snap648
			d154 = snap649
			d155 = snap650
			d156 = snap651
			d157 = snap652
			d158 = snap653
			d159 = snap654
			d160 = snap655
			d161 = snap656
			d162 = snap657
			d163 = snap658
			d164 = snap659
			d165 = snap660
			d166 = snap661
			d167 = snap662
			d168 = snap663
			d169 = snap664
			d170 = snap665
			d171 = snap666
			d172 = snap667
			d173 = snap668
			d174 = snap669
			d175 = snap670
			d176 = snap671
			d177 = snap672
			d178 = snap673
			d179 = snap674
			d180 = snap675
			d181 = snap676
			d182 = snap677
			d183 = snap678
			d184 = snap679
			d185 = snap680
			d186 = snap681
			d187 = snap682
			d188 = snap683
			d189 = snap684
			d190 = snap685
			d191 = snap686
			d192 = snap687
			d193 = snap688
			d194 = snap689
			d195 = snap690
			d196 = snap691
			d197 = snap692
			d198 = snap693
			d199 = snap694
			d200 = snap695
			d201 = snap696
			d202 = snap697
			d203 = snap698
			d204 = snap699
			d205 = snap700
			d206 = snap701
			d207 = snap702
			d208 = snap703
			d209 = snap704
			d210 = snap705
			d211 = snap706
			d212 = snap707
			d213 = snap708
			d214 = snap709
			d215 = snap710
			d216 = snap711
			d217 = snap712
			d218 = snap713
			d219 = snap714
			d220 = snap715
			d221 = snap716
			d222 = snap717
			d223 = snap718
			d224 = snap719
			d225 = snap720
			d226 = snap721
			d227 = snap722
			d228 = snap723
			d229 = snap724
			d230 = snap725
			d231 = snap726
			d232 = snap727
			d233 = snap728
			d234 = snap729
			d235 = snap730
			d236 = snap731
			d237 = snap732
			d238 = snap733
			d239 = snap734
			d240 = snap735
			d487 = snap736
			d488 = snap737
			d489 = snap738
			d490 = snap739
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps493)
			}
			return result
			ctx.FreeDesc(&d488)
			return result
			}
			bbs[3].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[3].VisitCount >= 2 {
					ps.General = true
					return bbs[3].RenderPS(ps)
				}
			}
			bbs[3].VisitCount++
			if ps.General {
				if bbs[3].Rendered {
					ctx.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
				d47 = ps.OverlayValues[47]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
				d108 = ps.OverlayValues[108]
			}
			if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
				d109 = ps.OverlayValues[109]
			}
			if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
				d110 = ps.OverlayValues[110]
			}
			if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
				d111 = ps.OverlayValues[111]
			}
			if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
				d112 = ps.OverlayValues[112]
			}
			if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
				d113 = ps.OverlayValues[113]
			}
			if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
				d114 = ps.OverlayValues[114]
			}
			if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
				d115 = ps.OverlayValues[115]
			}
			if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
				d116 = ps.OverlayValues[116]
			}
			if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
				d117 = ps.OverlayValues[117]
			}
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != scm.LocNone {
				d118 = ps.OverlayValues[118]
			}
			if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
				d119 = ps.OverlayValues[119]
			}
			if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != scm.LocNone {
				d120 = ps.OverlayValues[120]
			}
			if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != scm.LocNone {
				d121 = ps.OverlayValues[121]
			}
			if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != scm.LocNone {
				d122 = ps.OverlayValues[122]
			}
			if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != scm.LocNone {
				d123 = ps.OverlayValues[123]
			}
			if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != scm.LocNone {
				d124 = ps.OverlayValues[124]
			}
			if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
				d125 = ps.OverlayValues[125]
			}
			if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
				d126 = ps.OverlayValues[126]
			}
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
				d128 = ps.OverlayValues[128]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
				d129 = ps.OverlayValues[129]
			}
			if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
				d130 = ps.OverlayValues[130]
			}
			if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
				d131 = ps.OverlayValues[131]
			}
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
				d133 = ps.OverlayValues[133]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
				d136 = ps.OverlayValues[136]
			}
			if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
				d137 = ps.OverlayValues[137]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
				d138 = ps.OverlayValues[138]
			}
			if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
				d139 = ps.OverlayValues[139]
			}
			if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
				d140 = ps.OverlayValues[140]
			}
			if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
				d141 = ps.OverlayValues[141]
			}
			if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
				d142 = ps.OverlayValues[142]
			}
			if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
				d143 = ps.OverlayValues[143]
			}
			if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
				d144 = ps.OverlayValues[144]
			}
			if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
				d145 = ps.OverlayValues[145]
			}
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
				d146 = ps.OverlayValues[146]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
				d155 = ps.OverlayValues[155]
			}
			if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
				d156 = ps.OverlayValues[156]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
				d159 = ps.OverlayValues[159]
			}
			if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
				d160 = ps.OverlayValues[160]
			}
			if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
				d161 = ps.OverlayValues[161]
			}
			if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
				d162 = ps.OverlayValues[162]
			}
			if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
				d163 = ps.OverlayValues[163]
			}
			if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
				d164 = ps.OverlayValues[164]
			}
			if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
				d165 = ps.OverlayValues[165]
			}
			if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
				d166 = ps.OverlayValues[166]
			}
			if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
				d167 = ps.OverlayValues[167]
			}
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
				d169 = ps.OverlayValues[169]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != scm.LocNone {
				d172 = ps.OverlayValues[172]
			}
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
				d176 = ps.OverlayValues[176]
			}
			if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
				d177 = ps.OverlayValues[177]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
				d179 = ps.OverlayValues[179]
			}
			if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
				d180 = ps.OverlayValues[180]
			}
			if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
				d181 = ps.OverlayValues[181]
			}
			if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
				d182 = ps.OverlayValues[182]
			}
			if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
				d183 = ps.OverlayValues[183]
			}
			if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
				d184 = ps.OverlayValues[184]
			}
			if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
				d185 = ps.OverlayValues[185]
			}
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
				d191 = ps.OverlayValues[191]
			}
			if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
				d192 = ps.OverlayValues[192]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
				d194 = ps.OverlayValues[194]
			}
			if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
				d195 = ps.OverlayValues[195]
			}
			if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
				d196 = ps.OverlayValues[196]
			}
			if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
				d197 = ps.OverlayValues[197]
			}
			if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
				d198 = ps.OverlayValues[198]
			}
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
				d205 = ps.OverlayValues[205]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != scm.LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != scm.LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != scm.LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != scm.LocNone {
				d211 = ps.OverlayValues[211]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != scm.LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != scm.LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != scm.LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != scm.LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != scm.LocNone {
				d216 = ps.OverlayValues[216]
			}
			if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != scm.LocNone {
				d217 = ps.OverlayValues[217]
			}
			if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != scm.LocNone {
				d218 = ps.OverlayValues[218]
			}
			if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != scm.LocNone {
				d219 = ps.OverlayValues[219]
			}
			if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
				d220 = ps.OverlayValues[220]
			}
			if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != scm.LocNone {
				d221 = ps.OverlayValues[221]
			}
			if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != scm.LocNone {
				d222 = ps.OverlayValues[222]
			}
			if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
				d223 = ps.OverlayValues[223]
			}
			if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != scm.LocNone {
				d224 = ps.OverlayValues[224]
			}
			if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
				d225 = ps.OverlayValues[225]
			}
			if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
				d226 = ps.OverlayValues[226]
			}
			if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
				d227 = ps.OverlayValues[227]
			}
			if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
				d228 = ps.OverlayValues[228]
			}
			if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
				d229 = ps.OverlayValues[229]
			}
			if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
				d230 = ps.OverlayValues[230]
			}
			if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
				d231 = ps.OverlayValues[231]
			}
			if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
				d232 = ps.OverlayValues[232]
			}
			if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
				d233 = ps.OverlayValues[233]
			}
			if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
				d234 = ps.OverlayValues[234]
			}
			if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
				d235 = ps.OverlayValues[235]
			}
			if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
				d236 = ps.OverlayValues[236]
			}
			if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
				d237 = ps.OverlayValues[237]
			}
			if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
				d238 = ps.OverlayValues[238]
			}
			if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
				d239 = ps.OverlayValues[239]
			}
			if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
				d240 = ps.OverlayValues[240]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			ctx.ReclaimUntrackedRegs()
			d741 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewString("invalid value in prefix storage")}
			ctx.EnsureDesc(&d741)
			ctx.EnsureDesc(&d741)
			if d741.Loc == scm.LocImm {
				tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d741.Imm.GetTag() == scm.TagBool {
					ctx.EmitMakeBool(tmpPair, d741)
				} else if d741.Imm.GetTag() == scm.TagInt {
					ctx.EmitMakeInt(tmpPair, d741)
				} else if d741.Imm.GetTag() == scm.TagFloat {
					ctx.EmitMakeFloat(tmpPair, d741)
				} else if d741.Imm.GetTag() == scm.TagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d741.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d741 = tmpPair
			} else if d741.Loc == scm.LocReg {
				tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: d741.Type, Reg: ctx.AllocRegExcept(d741.Reg), Reg2: ctx.AllocRegExcept(d741.Reg)}
				switch d741.Type {
				case scm.TagBool:
					ctx.EmitMakeBool(tmpPair, d741)
				case scm.TagInt:
					ctx.EmitMakeInt(tmpPair, d741)
				case scm.TagFloat:
					ctx.EmitMakeFloat(tmpPair, d741)
				default:
					panic("jit: panic arg scalar type unknown for scm.Scmer pair")
				}
				ctx.FreeDesc(&d741)
				d741 = tmpPair
			}
			if d741.Loc != scm.LocRegPair && d741.Loc != scm.LocStackPair {
				panic("jit: panic arg expects scm.Scmer pair")
			}
			ctx.EmitGoCallVoid(scm.GoFuncAddr(scm.JITPanic), []scm.JITValueDesc{d741})
			ctx.FreeDesc(&d741)
			return result
			}
			bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[4].VisitCount >= 2 {
					ps.General = true
					return bbs[4].RenderPS(ps)
				}
			}
			bbs[4].VisitCount++
			if ps.General {
				if bbs[4].Rendered {
					ctx.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
				d47 = ps.OverlayValues[47]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
				d108 = ps.OverlayValues[108]
			}
			if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
				d109 = ps.OverlayValues[109]
			}
			if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
				d110 = ps.OverlayValues[110]
			}
			if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
				d111 = ps.OverlayValues[111]
			}
			if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
				d112 = ps.OverlayValues[112]
			}
			if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
				d113 = ps.OverlayValues[113]
			}
			if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
				d114 = ps.OverlayValues[114]
			}
			if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
				d115 = ps.OverlayValues[115]
			}
			if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
				d116 = ps.OverlayValues[116]
			}
			if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
				d117 = ps.OverlayValues[117]
			}
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != scm.LocNone {
				d118 = ps.OverlayValues[118]
			}
			if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
				d119 = ps.OverlayValues[119]
			}
			if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != scm.LocNone {
				d120 = ps.OverlayValues[120]
			}
			if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != scm.LocNone {
				d121 = ps.OverlayValues[121]
			}
			if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != scm.LocNone {
				d122 = ps.OverlayValues[122]
			}
			if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != scm.LocNone {
				d123 = ps.OverlayValues[123]
			}
			if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != scm.LocNone {
				d124 = ps.OverlayValues[124]
			}
			if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
				d125 = ps.OverlayValues[125]
			}
			if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
				d126 = ps.OverlayValues[126]
			}
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
				d128 = ps.OverlayValues[128]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
				d129 = ps.OverlayValues[129]
			}
			if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
				d130 = ps.OverlayValues[130]
			}
			if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
				d131 = ps.OverlayValues[131]
			}
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
				d133 = ps.OverlayValues[133]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
				d136 = ps.OverlayValues[136]
			}
			if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
				d137 = ps.OverlayValues[137]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
				d138 = ps.OverlayValues[138]
			}
			if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
				d139 = ps.OverlayValues[139]
			}
			if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
				d140 = ps.OverlayValues[140]
			}
			if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
				d141 = ps.OverlayValues[141]
			}
			if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
				d142 = ps.OverlayValues[142]
			}
			if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
				d143 = ps.OverlayValues[143]
			}
			if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
				d144 = ps.OverlayValues[144]
			}
			if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
				d145 = ps.OverlayValues[145]
			}
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
				d146 = ps.OverlayValues[146]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
				d155 = ps.OverlayValues[155]
			}
			if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
				d156 = ps.OverlayValues[156]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
				d159 = ps.OverlayValues[159]
			}
			if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
				d160 = ps.OverlayValues[160]
			}
			if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
				d161 = ps.OverlayValues[161]
			}
			if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
				d162 = ps.OverlayValues[162]
			}
			if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
				d163 = ps.OverlayValues[163]
			}
			if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
				d164 = ps.OverlayValues[164]
			}
			if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
				d165 = ps.OverlayValues[165]
			}
			if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
				d166 = ps.OverlayValues[166]
			}
			if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
				d167 = ps.OverlayValues[167]
			}
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
				d169 = ps.OverlayValues[169]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != scm.LocNone {
				d172 = ps.OverlayValues[172]
			}
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
				d176 = ps.OverlayValues[176]
			}
			if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
				d177 = ps.OverlayValues[177]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
				d179 = ps.OverlayValues[179]
			}
			if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
				d180 = ps.OverlayValues[180]
			}
			if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
				d181 = ps.OverlayValues[181]
			}
			if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
				d182 = ps.OverlayValues[182]
			}
			if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
				d183 = ps.OverlayValues[183]
			}
			if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
				d184 = ps.OverlayValues[184]
			}
			if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
				d185 = ps.OverlayValues[185]
			}
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
				d191 = ps.OverlayValues[191]
			}
			if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
				d192 = ps.OverlayValues[192]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
				d194 = ps.OverlayValues[194]
			}
			if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
				d195 = ps.OverlayValues[195]
			}
			if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
				d196 = ps.OverlayValues[196]
			}
			if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
				d197 = ps.OverlayValues[197]
			}
			if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
				d198 = ps.OverlayValues[198]
			}
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
				d205 = ps.OverlayValues[205]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != scm.LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != scm.LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != scm.LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != scm.LocNone {
				d211 = ps.OverlayValues[211]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != scm.LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != scm.LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != scm.LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != scm.LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != scm.LocNone {
				d216 = ps.OverlayValues[216]
			}
			if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != scm.LocNone {
				d217 = ps.OverlayValues[217]
			}
			if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != scm.LocNone {
				d218 = ps.OverlayValues[218]
			}
			if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != scm.LocNone {
				d219 = ps.OverlayValues[219]
			}
			if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
				d220 = ps.OverlayValues[220]
			}
			if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != scm.LocNone {
				d221 = ps.OverlayValues[221]
			}
			if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != scm.LocNone {
				d222 = ps.OverlayValues[222]
			}
			if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
				d223 = ps.OverlayValues[223]
			}
			if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != scm.LocNone {
				d224 = ps.OverlayValues[224]
			}
			if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
				d225 = ps.OverlayValues[225]
			}
			if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
				d226 = ps.OverlayValues[226]
			}
			if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
				d227 = ps.OverlayValues[227]
			}
			if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
				d228 = ps.OverlayValues[228]
			}
			if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
				d229 = ps.OverlayValues[229]
			}
			if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
				d230 = ps.OverlayValues[230]
			}
			if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
				d231 = ps.OverlayValues[231]
			}
			if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
				d232 = ps.OverlayValues[232]
			}
			if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
				d233 = ps.OverlayValues[233]
			}
			if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
				d234 = ps.OverlayValues[234]
			}
			if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
				d235 = ps.OverlayValues[235]
			}
			if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
				d236 = ps.OverlayValues[236]
			}
			if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
				d237 = ps.OverlayValues[237]
			}
			if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
				d238 = ps.OverlayValues[238]
			}
			if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
				d239 = ps.OverlayValues[239]
			}
			if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
				d240 = ps.OverlayValues[240]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			if len(ps.OverlayValues) > 741 && ps.OverlayValues[741].Loc != scm.LocNone {
				d741 = ps.OverlayValues[741]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			d742 = idxInt
			_ = d742
			r246 := idxInt.Loc == scm.LocReg
			r247 := idxInt.Reg
			if r246 { ctx.ProtectReg(r247) }
			d743 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			lbl72 := ctx.ReserveLabel()
			bbpos_7_0 := int32(-1)
			_ = bbpos_7_0
			bbpos_7_1 := int32(-1)
			_ = bbpos_7_1
			bbpos_7_2 := int32(-1)
			_ = bbpos_7_2
			bbpos_7_3 := int32(-1)
			_ = bbpos_7_3
			bbpos_7_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d743 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			ctx.EnsureDesc(&d742)
			ctx.EnsureDesc(&d742)
			var d744 scm.JITValueDesc
			if d742.Loc == scm.LocImm {
				d744 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d742.Imm.Int()))))}
			} else {
				r248 := ctx.AllocReg()
				ctx.EmitMovRegReg(r248, d742.Reg)
				ctx.EmitShlRegImm8(r248, 32)
				ctx.EmitShrRegImm8(r248, 32)
				d744 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r248}
				ctx.BindReg(r248, &d744)
			}
			var d745 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d745 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r249 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r249, thisptr.Reg, off)
				d745 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r249}
				ctx.BindReg(r249, &d745)
			}
			ctx.EnsureDesc(&d745)
			ctx.EnsureDesc(&d745)
			var d746 scm.JITValueDesc
			if d745.Loc == scm.LocImm {
				d746 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d745.Imm.Int()))))}
			} else {
				r250 := ctx.AllocReg()
				ctx.EmitMovRegReg(r250, d745.Reg)
				ctx.EmitShlRegImm8(r250, 56)
				ctx.EmitShrRegImm8(r250, 56)
				d746 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r250}
				ctx.BindReg(r250, &d746)
			}
			ctx.FreeDesc(&d745)
			ctx.EnsureDesc(&d744)
			ctx.EnsureDesc(&d746)
			ctx.EnsureDesc(&d744)
			ctx.EnsureDesc(&d746)
			ctx.EnsureDesc(&d744)
			ctx.EnsureDesc(&d746)
			var d747 scm.JITValueDesc
			if d744.Loc == scm.LocImm && d746.Loc == scm.LocImm {
				d747 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d744.Imm.Int() * d746.Imm.Int())}
			} else if d744.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d746.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d744.Imm.Int()))
				ctx.EmitImulInt64(scratch, d746.Reg)
				d747 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d747)
			} else if d746.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d744.Reg)
				ctx.EmitMovRegReg(scratch, d744.Reg)
				if d746.Imm.Int() >= -2147483648 && d746.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d746.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d746.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d747 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d747)
			} else {
				r251 := ctx.AllocRegExcept(d744.Reg, d746.Reg)
				ctx.EmitMovRegReg(r251, d744.Reg)
				ctx.EmitImulInt64(r251, d746.Reg)
				d747 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r251}
				ctx.BindReg(r251, &d747)
			}
			if d747.Loc == scm.LocReg && d744.Loc == scm.LocReg && d747.Reg == d744.Reg {
				ctx.TransferReg(d744.Reg)
				d744.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d744)
			ctx.FreeDesc(&d746)
			var d748 scm.JITValueDesc
			r252 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r252, uint64(dataPtr))
				d748 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r252, StackOff: int32(sliceLen)}
				ctx.BindReg(r252, &d748)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0)
				ctx.EmitMovRegMem(r252, thisptr.Reg, off)
				d748 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r252}
				ctx.BindReg(r252, &d748)
			}
			ctx.BindReg(r252, &d748)
			ctx.EnsureDesc(&d747)
			var d749 scm.JITValueDesc
			if d747.Loc == scm.LocImm {
				d749 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d747.Imm.Int() / 64)}
			} else {
				r253 := ctx.AllocRegExcept(d747.Reg)
				ctx.EmitMovRegReg(r253, d747.Reg)
				ctx.EmitShrRegImm8(r253, 6)
				d749 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r253}
				ctx.BindReg(r253, &d749)
			}
			if d749.Loc == scm.LocReg && d747.Loc == scm.LocReg && d749.Reg == d747.Reg {
				ctx.TransferReg(d747.Reg)
				d747.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d749)
			r254 := ctx.AllocReg()
			ctx.EnsureDesc(&d749)
			ctx.EnsureDesc(&d748)
			if d749.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r254, uint64(d749.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r254, d749.Reg)
				ctx.EmitShlRegImm8(r254, 3)
			}
			if d748.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d748.Imm.Int()))
				ctx.EmitAddInt64(r254, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r254, d748.Reg)
			}
			r255 := ctx.AllocRegExcept(r254)
			ctx.EmitMovRegMem(r255, r254, 0)
			ctx.FreeReg(r254)
			d750 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r255}
			ctx.BindReg(r255, &d750)
			ctx.FreeDesc(&d749)
			ctx.EnsureDesc(&d747)
			var d751 scm.JITValueDesc
			if d747.Loc == scm.LocImm {
				d751 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d747.Imm.Int() % 64)}
			} else {
				r256 := ctx.AllocRegExcept(d747.Reg)
				ctx.EmitMovRegReg(r256, d747.Reg)
				ctx.EmitAndRegImm32(r256, 63)
				d751 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r256}
				ctx.BindReg(r256, &d751)
			}
			if d751.Loc == scm.LocReg && d747.Loc == scm.LocReg && d751.Reg == d747.Reg {
				ctx.TransferReg(d747.Reg)
				d747.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d750)
			ctx.EnsureDesc(&d751)
			var d752 scm.JITValueDesc
			if d750.Loc == scm.LocImm && d751.Loc == scm.LocImm {
				d752 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d750.Imm.Int()) << uint64(d751.Imm.Int())))}
			} else if d751.Loc == scm.LocImm {
				r257 := ctx.AllocRegExcept(d750.Reg)
				ctx.EmitMovRegReg(r257, d750.Reg)
				ctx.EmitShlRegImm8(r257, uint8(d751.Imm.Int()))
				d752 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r257}
				ctx.BindReg(r257, &d752)
			} else {
				{
					shiftSrc := d750.Reg
					r258 := ctx.AllocRegExcept(d750.Reg)
					ctx.EmitMovRegReg(r258, d750.Reg)
					shiftSrc = r258
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d751.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d751.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d751.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d752 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d752)
				}
			}
			if d752.Loc == scm.LocReg && d750.Loc == scm.LocReg && d752.Reg == d750.Reg {
				ctx.TransferReg(d750.Reg)
				d750.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d750)
			ctx.FreeDesc(&d751)
			var d753 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d753 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25)
				r259 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r259, thisptr.Reg, off)
				d753 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r259}
				ctx.BindReg(r259, &d753)
			}
			d754 = d753
			ctx.EnsureDesc(&d754)
			if d754.Loc != scm.LocImm && d754.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl73 := ctx.ReserveLabel()
			lbl74 := ctx.ReserveLabel()
			lbl75 := ctx.ReserveLabel()
			lbl76 := ctx.ReserveLabel()
			if d754.Loc == scm.LocImm {
				if d754.Imm.Bool() {
					ctx.MarkLabel(lbl75)
					ctx.EmitJmp(lbl73)
				} else {
					ctx.MarkLabel(lbl76)
			ctx.EnsureDesc(&d752)
			if d752.Loc == scm.LocReg {
				ctx.ProtectReg(d752.Reg)
			} else if d752.Loc == scm.LocRegPair {
				ctx.ProtectReg(d752.Reg)
				ctx.ProtectReg(d752.Reg2)
			}
			d755 = d752
			if d755.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d755)
			ctx.EmitStoreToStack(d755, int32(bbs[2].PhiBase)+int32(0))
			if d752.Loc == scm.LocReg {
				ctx.UnprotectReg(d752.Reg)
			} else if d752.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d752.Reg)
				ctx.UnprotectReg(d752.Reg2)
			}
					ctx.EmitJmp(lbl74)
				}
			} else {
				ctx.EmitCmpRegImm32(d754.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl75)
				ctx.EmitJmp(lbl76)
				ctx.MarkLabel(lbl75)
				ctx.EmitJmp(lbl73)
				ctx.MarkLabel(lbl76)
			ctx.EnsureDesc(&d752)
			if d752.Loc == scm.LocReg {
				ctx.ProtectReg(d752.Reg)
			} else if d752.Loc == scm.LocRegPair {
				ctx.ProtectReg(d752.Reg)
				ctx.ProtectReg(d752.Reg2)
			}
			d756 = d752
			if d756.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d756)
			ctx.EmitStoreToStack(d756, int32(bbs[2].PhiBase)+int32(0))
			if d752.Loc == scm.LocReg {
				ctx.UnprotectReg(d752.Reg)
			} else if d752.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d752.Reg)
				ctx.UnprotectReg(d752.Reg2)
			}
				ctx.EmitJmp(lbl74)
			}
			ctx.FreeDesc(&d753)
			bbpos_7_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl74)
			ctx.ResolveFixups()
			d743 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			var d757 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d757 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r260 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r260, thisptr.Reg, off)
				d757 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r260}
				ctx.BindReg(r260, &d757)
			}
			ctx.EnsureDesc(&d757)
			ctx.EnsureDesc(&d757)
			var d758 scm.JITValueDesc
			if d757.Loc == scm.LocImm {
				d758 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d757.Imm.Int()))))}
			} else {
				r261 := ctx.AllocReg()
				ctx.EmitMovRegReg(r261, d757.Reg)
				ctx.EmitShlRegImm8(r261, 56)
				ctx.EmitShrRegImm8(r261, 56)
				d758 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r261}
				ctx.BindReg(r261, &d758)
			}
			ctx.FreeDesc(&d757)
			d759 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d758)
			ctx.EnsureDesc(&d759)
			ctx.EnsureDesc(&d758)
			ctx.EnsureDesc(&d759)
			ctx.EnsureDesc(&d758)
			var d760 scm.JITValueDesc
			if d759.Loc == scm.LocImm && d758.Loc == scm.LocImm {
				d760 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d759.Imm.Int() - d758.Imm.Int())}
			} else if d758.Loc == scm.LocImm && d758.Imm.Int() == 0 {
				r262 := ctx.AllocRegExcept(d759.Reg)
				ctx.EmitMovRegReg(r262, d759.Reg)
				d760 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r262}
				ctx.BindReg(r262, &d760)
			} else if d759.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d758.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d759.Imm.Int()))
				ctx.EmitSubInt64(scratch, d758.Reg)
				d760 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d760)
			} else if d758.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d759.Reg)
				ctx.EmitMovRegReg(scratch, d759.Reg)
				if d758.Imm.Int() >= -2147483648 && d758.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d758.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d758.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d760 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d760)
			} else {
				r263 := ctx.AllocRegExcept(d759.Reg, d758.Reg)
				ctx.EmitMovRegReg(r263, d759.Reg)
				ctx.EmitSubInt64(r263, d758.Reg)
				d760 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r263}
				ctx.BindReg(r263, &d760)
			}
			if d760.Loc == scm.LocReg && d759.Loc == scm.LocReg && d760.Reg == d759.Reg {
				ctx.TransferReg(d759.Reg)
				d759.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d758)
			ctx.EnsureDesc(&d743)
			ctx.EnsureDesc(&d760)
			var d761 scm.JITValueDesc
			if d743.Loc == scm.LocImm && d760.Loc == scm.LocImm {
				d761 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d743.Imm.Int()) >> uint64(d760.Imm.Int())))}
			} else if d760.Loc == scm.LocImm {
				r264 := ctx.AllocRegExcept(d743.Reg)
				ctx.EmitMovRegReg(r264, d743.Reg)
				ctx.EmitShrRegImm8(r264, uint8(d760.Imm.Int()))
				d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r264}
				ctx.BindReg(r264, &d761)
			} else {
				{
					shiftSrc := d743.Reg
					r265 := ctx.AllocRegExcept(d743.Reg)
					ctx.EmitMovRegReg(r265, d743.Reg)
					shiftSrc = r265
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d760.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d760.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d760.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d761)
				}
			}
			if d761.Loc == scm.LocReg && d743.Loc == scm.LocReg && d761.Reg == d743.Reg {
				ctx.TransferReg(d743.Reg)
				d743.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d743)
			ctx.FreeDesc(&d760)
			r266 := ctx.AllocReg()
			ctx.EnsureDesc(&d761)
			ctx.EnsureDesc(&d761)
			if d761.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r266, d761)
			}
			ctx.EmitJmp(lbl72)
			bbpos_7_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl73)
			ctx.ResolveFixups()
			d743 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			ctx.EnsureDesc(&d747)
			var d762 scm.JITValueDesc
			if d747.Loc == scm.LocImm {
				d762 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d747.Imm.Int() % 64)}
			} else {
				r267 := ctx.AllocRegExcept(d747.Reg)
				ctx.EmitMovRegReg(r267, d747.Reg)
				ctx.EmitAndRegImm32(r267, 63)
				d762 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r267}
				ctx.BindReg(r267, &d762)
			}
			if d762.Loc == scm.LocReg && d747.Loc == scm.LocReg && d762.Reg == d747.Reg {
				ctx.TransferReg(d747.Reg)
				d747.Loc = scm.LocNone
			}
			var d763 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d763 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r268 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r268, thisptr.Reg, off)
				d763 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r268}
				ctx.BindReg(r268, &d763)
			}
			ctx.EnsureDesc(&d763)
			ctx.EnsureDesc(&d763)
			var d764 scm.JITValueDesc
			if d763.Loc == scm.LocImm {
				d764 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d763.Imm.Int()))))}
			} else {
				r269 := ctx.AllocReg()
				ctx.EmitMovRegReg(r269, d763.Reg)
				ctx.EmitShlRegImm8(r269, 56)
				ctx.EmitShrRegImm8(r269, 56)
				d764 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r269}
				ctx.BindReg(r269, &d764)
			}
			ctx.FreeDesc(&d763)
			ctx.EnsureDesc(&d762)
			ctx.EnsureDesc(&d764)
			ctx.EnsureDesc(&d762)
			ctx.EnsureDesc(&d764)
			ctx.EnsureDesc(&d762)
			ctx.EnsureDesc(&d764)
			var d765 scm.JITValueDesc
			if d762.Loc == scm.LocImm && d764.Loc == scm.LocImm {
				d765 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d762.Imm.Int() + d764.Imm.Int())}
			} else if d764.Loc == scm.LocImm && d764.Imm.Int() == 0 {
				r270 := ctx.AllocRegExcept(d762.Reg)
				ctx.EmitMovRegReg(r270, d762.Reg)
				d765 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r270}
				ctx.BindReg(r270, &d765)
			} else if d762.Loc == scm.LocImm && d762.Imm.Int() == 0 {
				d765 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d764.Reg}
				ctx.BindReg(d764.Reg, &d765)
			} else if d762.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d764.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d762.Imm.Int()))
				ctx.EmitAddInt64(scratch, d764.Reg)
				d765 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d765)
			} else if d764.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d762.Reg)
				ctx.EmitMovRegReg(scratch, d762.Reg)
				if d764.Imm.Int() >= -2147483648 && d764.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d764.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d764.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d765 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d765)
			} else {
				r271 := ctx.AllocRegExcept(d762.Reg, d764.Reg)
				ctx.EmitMovRegReg(r271, d762.Reg)
				ctx.EmitAddInt64(r271, d764.Reg)
				d765 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r271}
				ctx.BindReg(r271, &d765)
			}
			if d765.Loc == scm.LocReg && d762.Loc == scm.LocReg && d765.Reg == d762.Reg {
				ctx.TransferReg(d762.Reg)
				d762.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d762)
			ctx.FreeDesc(&d764)
			ctx.EnsureDesc(&d765)
			var d766 scm.JITValueDesc
			if d765.Loc == scm.LocImm {
				d766 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d765.Imm.Int()) > uint64(64))}
			} else {
				r272 := ctx.AllocRegExcept(d765.Reg)
				ctx.EmitCmpRegImm32(d765.Reg, 64)
				ctx.EmitSetcc(r272, scm.CcA)
				d766 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r272}
				ctx.BindReg(r272, &d766)
			}
			ctx.FreeDesc(&d765)
			d767 = d766
			ctx.EnsureDesc(&d767)
			if d767.Loc != scm.LocImm && d767.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl77 := ctx.ReserveLabel()
			lbl78 := ctx.ReserveLabel()
			lbl79 := ctx.ReserveLabel()
			if d767.Loc == scm.LocImm {
				if d767.Imm.Bool() {
					ctx.MarkLabel(lbl78)
					ctx.EmitJmp(lbl77)
				} else {
					ctx.MarkLabel(lbl79)
			ctx.EnsureDesc(&d752)
			if d752.Loc == scm.LocReg {
				ctx.ProtectReg(d752.Reg)
			} else if d752.Loc == scm.LocRegPair {
				ctx.ProtectReg(d752.Reg)
				ctx.ProtectReg(d752.Reg2)
			}
			d768 = d752
			if d768.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d768)
			ctx.EmitStoreToStack(d768, int32(bbs[2].PhiBase)+int32(0))
			if d752.Loc == scm.LocReg {
				ctx.UnprotectReg(d752.Reg)
			} else if d752.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d752.Reg)
				ctx.UnprotectReg(d752.Reg2)
			}
					ctx.EmitJmp(lbl74)
				}
			} else {
				ctx.EmitCmpRegImm32(d767.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl78)
				ctx.EmitJmp(lbl79)
				ctx.MarkLabel(lbl78)
				ctx.EmitJmp(lbl77)
				ctx.MarkLabel(lbl79)
			ctx.EnsureDesc(&d752)
			if d752.Loc == scm.LocReg {
				ctx.ProtectReg(d752.Reg)
			} else if d752.Loc == scm.LocRegPair {
				ctx.ProtectReg(d752.Reg)
				ctx.ProtectReg(d752.Reg2)
			}
			d769 = d752
			if d769.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d769)
			ctx.EmitStoreToStack(d769, int32(bbs[2].PhiBase)+int32(0))
			if d752.Loc == scm.LocReg {
				ctx.UnprotectReg(d752.Reg)
			} else if d752.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d752.Reg)
				ctx.UnprotectReg(d752.Reg2)
			}
				ctx.EmitJmp(lbl74)
			}
			ctx.FreeDesc(&d766)
			bbpos_7_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl77)
			ctx.ResolveFixups()
			d743 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			ctx.EnsureDesc(&d747)
			var d770 scm.JITValueDesc
			if d747.Loc == scm.LocImm {
				d770 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d747.Imm.Int() / 64)}
			} else {
				r273 := ctx.AllocRegExcept(d747.Reg)
				ctx.EmitMovRegReg(r273, d747.Reg)
				ctx.EmitShrRegImm8(r273, 6)
				d770 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r273}
				ctx.BindReg(r273, &d770)
			}
			if d770.Loc == scm.LocReg && d747.Loc == scm.LocReg && d770.Reg == d747.Reg {
				ctx.TransferReg(d747.Reg)
				d747.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d770)
			ctx.EnsureDesc(&d770)
			var d771 scm.JITValueDesc
			if d770.Loc == scm.LocImm {
				d771 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d770.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d770.Reg)
				ctx.EmitMovRegReg(scratch, d770.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d771 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d771)
			}
			if d771.Loc == scm.LocReg && d770.Loc == scm.LocReg && d771.Reg == d770.Reg {
				ctx.TransferReg(d770.Reg)
				d770.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d770)
			ctx.EnsureDesc(&d771)
			r274 := ctx.AllocReg()
			ctx.EnsureDesc(&d771)
			ctx.EnsureDesc(&d748)
			if d771.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r274, uint64(d771.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r274, d771.Reg)
				ctx.EmitShlRegImm8(r274, 3)
			}
			if d748.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d748.Imm.Int()))
				ctx.EmitAddInt64(r274, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r274, d748.Reg)
			}
			r275 := ctx.AllocRegExcept(r274)
			ctx.EmitMovRegMem(r275, r274, 0)
			ctx.FreeReg(r274)
			d772 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r275}
			ctx.BindReg(r275, &d772)
			ctx.FreeDesc(&d771)
			ctx.EnsureDesc(&d747)
			var d773 scm.JITValueDesc
			if d747.Loc == scm.LocImm {
				d773 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d747.Imm.Int() % 64)}
			} else {
				r276 := ctx.AllocRegExcept(d747.Reg)
				ctx.EmitMovRegReg(r276, d747.Reg)
				ctx.EmitAndRegImm32(r276, 63)
				d773 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r276}
				ctx.BindReg(r276, &d773)
			}
			if d773.Loc == scm.LocReg && d747.Loc == scm.LocReg && d773.Reg == d747.Reg {
				ctx.TransferReg(d747.Reg)
				d747.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d747)
			d774 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d773)
			ctx.EnsureDesc(&d774)
			ctx.EnsureDesc(&d773)
			ctx.EnsureDesc(&d774)
			ctx.EnsureDesc(&d773)
			var d775 scm.JITValueDesc
			if d774.Loc == scm.LocImm && d773.Loc == scm.LocImm {
				d775 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d774.Imm.Int() - d773.Imm.Int())}
			} else if d773.Loc == scm.LocImm && d773.Imm.Int() == 0 {
				r277 := ctx.AllocRegExcept(d774.Reg)
				ctx.EmitMovRegReg(r277, d774.Reg)
				d775 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r277}
				ctx.BindReg(r277, &d775)
			} else if d774.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d773.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d774.Imm.Int()))
				ctx.EmitSubInt64(scratch, d773.Reg)
				d775 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d775)
			} else if d773.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d774.Reg)
				ctx.EmitMovRegReg(scratch, d774.Reg)
				if d773.Imm.Int() >= -2147483648 && d773.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d773.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d773.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d775 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d775)
			} else {
				r278 := ctx.AllocRegExcept(d774.Reg, d773.Reg)
				ctx.EmitMovRegReg(r278, d774.Reg)
				ctx.EmitSubInt64(r278, d773.Reg)
				d775 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r278}
				ctx.BindReg(r278, &d775)
			}
			if d775.Loc == scm.LocReg && d774.Loc == scm.LocReg && d775.Reg == d774.Reg {
				ctx.TransferReg(d774.Reg)
				d774.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d773)
			ctx.EnsureDesc(&d772)
			ctx.EnsureDesc(&d775)
			var d776 scm.JITValueDesc
			if d772.Loc == scm.LocImm && d775.Loc == scm.LocImm {
				d776 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d772.Imm.Int()) >> uint64(d775.Imm.Int())))}
			} else if d775.Loc == scm.LocImm {
				r279 := ctx.AllocRegExcept(d772.Reg)
				ctx.EmitMovRegReg(r279, d772.Reg)
				ctx.EmitShrRegImm8(r279, uint8(d775.Imm.Int()))
				d776 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r279}
				ctx.BindReg(r279, &d776)
			} else {
				{
					shiftSrc := d772.Reg
					r280 := ctx.AllocRegExcept(d772.Reg)
					ctx.EmitMovRegReg(r280, d772.Reg)
					shiftSrc = r280
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d775.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d775.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d775.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d776 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d776)
				}
			}
			if d776.Loc == scm.LocReg && d772.Loc == scm.LocReg && d776.Reg == d772.Reg {
				ctx.TransferReg(d772.Reg)
				d772.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d772)
			ctx.FreeDesc(&d775)
			ctx.EnsureDesc(&d752)
			ctx.EnsureDesc(&d776)
			var d777 scm.JITValueDesc
			if d752.Loc == scm.LocImm && d776.Loc == scm.LocImm {
				d777 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d752.Imm.Int() | d776.Imm.Int())}
			} else if d752.Loc == scm.LocImm && d752.Imm.Int() == 0 {
				d777 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d776.Reg}
				ctx.BindReg(d776.Reg, &d777)
			} else if d776.Loc == scm.LocImm && d776.Imm.Int() == 0 {
				r281 := ctx.AllocRegExcept(d752.Reg)
				ctx.EmitMovRegReg(r281, d752.Reg)
				d777 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r281}
				ctx.BindReg(r281, &d777)
			} else if d752.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d776.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d752.Imm.Int()))
				ctx.EmitOrInt64(scratch, d776.Reg)
				d777 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d777)
			} else if d776.Loc == scm.LocImm {
				r282 := ctx.AllocRegExcept(d752.Reg)
				ctx.EmitMovRegReg(r282, d752.Reg)
				if d776.Imm.Int() >= -2147483648 && d776.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r282, int32(d776.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d776.Imm.Int()))
					ctx.EmitOrInt64(r282, scm.RegR11)
				}
				d777 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r282}
				ctx.BindReg(r282, &d777)
			} else {
				r283 := ctx.AllocRegExcept(d752.Reg, d776.Reg)
				ctx.EmitMovRegReg(r283, d752.Reg)
				ctx.EmitOrInt64(r283, d776.Reg)
				d777 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r283}
				ctx.BindReg(r283, &d777)
			}
			if d777.Loc == scm.LocReg && d752.Loc == scm.LocReg && d777.Reg == d752.Reg {
				ctx.TransferReg(d752.Reg)
				d752.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d776)
			ctx.EnsureDesc(&d777)
			if d777.Loc == scm.LocReg {
				ctx.ProtectReg(d777.Reg)
			} else if d777.Loc == scm.LocRegPair {
				ctx.ProtectReg(d777.Reg)
				ctx.ProtectReg(d777.Reg2)
			}
			d778 = d777
			if d778.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d778)
			ctx.EmitStoreToStack(d778, int32(bbs[2].PhiBase)+int32(0))
			if d777.Loc == scm.LocReg {
				ctx.UnprotectReg(d777.Reg)
			} else if d777.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d777.Reg)
				ctx.UnprotectReg(d777.Reg2)
			}
			ctx.EmitJmp(lbl74)
			ctx.MarkLabel(lbl72)
			d779 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r266}
			ctx.BindReg(r266, &d779)
			ctx.BindReg(r266, &d779)
			if r246 { ctx.UnprotectReg(r247) }
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d779)
			ctx.EnsureDesc(&d779)
			var d780 scm.JITValueDesc
			if d779.Loc == scm.LocImm {
				d780 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d779.Imm.Int()))))}
			} else {
				r284 := ctx.AllocReg()
				ctx.EmitMovRegReg(r284, d779.Reg)
				d780 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r284}
				ctx.BindReg(r284, &d780)
			}
			ctx.FreeDesc(&d779)
			var d781 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d781 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32)
				r285 := ctx.AllocReg()
				ctx.EmitMovRegMem(r285, thisptr.Reg, off)
				d781 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r285}
				ctx.BindReg(r285, &d781)
			}
			ctx.EnsureDesc(&d780)
			ctx.EnsureDesc(&d781)
			ctx.EnsureDesc(&d780)
			ctx.EnsureDesc(&d781)
			ctx.EnsureDesc(&d780)
			ctx.EnsureDesc(&d781)
			var d782 scm.JITValueDesc
			if d780.Loc == scm.LocImm && d781.Loc == scm.LocImm {
				d782 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d780.Imm.Int() + d781.Imm.Int())}
			} else if d781.Loc == scm.LocImm && d781.Imm.Int() == 0 {
				r286 := ctx.AllocRegExcept(d780.Reg)
				ctx.EmitMovRegReg(r286, d780.Reg)
				d782 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r286}
				ctx.BindReg(r286, &d782)
			} else if d780.Loc == scm.LocImm && d780.Imm.Int() == 0 {
				d782 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d781.Reg}
				ctx.BindReg(d781.Reg, &d782)
			} else if d780.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d781.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d780.Imm.Int()))
				ctx.EmitAddInt64(scratch, d781.Reg)
				d782 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d782)
			} else if d781.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d780.Reg)
				ctx.EmitMovRegReg(scratch, d780.Reg)
				if d781.Imm.Int() >= -2147483648 && d781.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d781.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d781.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d782 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d782)
			} else {
				r287 := ctx.AllocRegExcept(d780.Reg, d781.Reg)
				ctx.EmitMovRegReg(r287, d780.Reg)
				ctx.EmitAddInt64(r287, d781.Reg)
				d782 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r287}
				ctx.BindReg(r287, &d782)
			}
			if d782.Loc == scm.LocReg && d780.Loc == scm.LocReg && d782.Reg == d780.Reg {
				ctx.TransferReg(d780.Reg)
				d780.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d780)
			ctx.FreeDesc(&d781)
			var d783 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
				r288 := ctx.AllocReg()
				r289 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r288, fieldAddr)
				ctx.EmitMovRegMem64(r289, fieldAddr+8)
				d783 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r288, Reg2: r289}
				ctx.BindReg(r288, &d783)
				ctx.BindReg(r289, &d783)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
				r290 := ctx.AllocReg()
				r291 := ctx.AllocReg()
				ctx.EmitMovRegMem(r290, thisptr.Reg, off)
				ctx.EmitMovRegMem(r291, thisptr.Reg, off+8)
				d783 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r290, Reg2: r291}
				ctx.BindReg(r290, &d783)
				ctx.BindReg(r291, &d783)
			}
			var d784 scm.JITValueDesc
			if d783.Loc == scm.LocImm {
				d784 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d783.StackOff))}
			} else {
				ctx.EnsureDesc(&d783)
				if d783.Loc == scm.LocRegPair {
					d784 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d783.Reg2}
					ctx.BindReg(d783.Reg2, &d784)
					ctx.BindReg(d783.Reg2, &d784)
				} else if d783.Loc == scm.LocReg {
					d784 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d783.Reg}
					ctx.BindReg(d783.Reg, &d784)
					ctx.BindReg(d783.Reg, &d784)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d784)
			ctx.EnsureDesc(&d784)
			ctx.EnsureDesc(&d782)
			ctx.EnsureDesc(&d784)
			ctx.EnsureDesc(&d782)
			ctx.EnsureDesc(&d784)
			ctx.EnsureDesc(&d782)
			ctx.EnsureDesc(&d784)
			var d786 scm.JITValueDesc
			if d782.Loc == scm.LocImm && d784.Loc == scm.LocImm {
				d786 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d782.Imm.Int() >= d784.Imm.Int())}
			} else if d784.Loc == scm.LocImm {
				r292 := ctx.AllocRegExcept(d782.Reg)
				if d784.Imm.Int() >= -2147483648 && d784.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d782.Reg, int32(d784.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d784.Imm.Int()))
					ctx.EmitCmpInt64(d782.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r292, scm.CcGE)
				d786 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r292}
				ctx.BindReg(r292, &d786)
			} else if d782.Loc == scm.LocImm {
				r293 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d782.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d784.Reg)
				ctx.EmitSetcc(r293, scm.CcGE)
				d786 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r293}
				ctx.BindReg(r293, &d786)
			} else {
				r294 := ctx.AllocRegExcept(d782.Reg)
				ctx.EmitCmpInt64(d782.Reg, d784.Reg)
				ctx.EmitSetcc(r294, scm.CcGE)
				d786 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r294}
				ctx.BindReg(r294, &d786)
			}
			ctx.FreeDesc(&d784)
			d787 = d786
			ctx.EnsureDesc(&d787)
			if d787.Loc != scm.LocImm && d787.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d787.Loc == scm.LocImm {
				if d787.Imm.Bool() {
			ps788 := scm.PhiState{General: ps.General}
			ps788.OverlayValues = make([]scm.JITValueDesc, 788)
			ps788.OverlayValues[0] = d0
			ps788.OverlayValues[1] = d1
			ps788.OverlayValues[2] = d2
			ps788.OverlayValues[3] = d3
			ps788.OverlayValues[4] = d4
			ps788.OverlayValues[5] = d5
			ps788.OverlayValues[6] = d6
			ps788.OverlayValues[7] = d7
			ps788.OverlayValues[8] = d8
			ps788.OverlayValues[9] = d9
			ps788.OverlayValues[10] = d10
			ps788.OverlayValues[11] = d11
			ps788.OverlayValues[12] = d12
			ps788.OverlayValues[13] = d13
			ps788.OverlayValues[14] = d14
			ps788.OverlayValues[15] = d15
			ps788.OverlayValues[16] = d16
			ps788.OverlayValues[17] = d17
			ps788.OverlayValues[18] = d18
			ps788.OverlayValues[19] = d19
			ps788.OverlayValues[20] = d20
			ps788.OverlayValues[21] = d21
			ps788.OverlayValues[22] = d22
			ps788.OverlayValues[23] = d23
			ps788.OverlayValues[24] = d24
			ps788.OverlayValues[25] = d25
			ps788.OverlayValues[26] = d26
			ps788.OverlayValues[27] = d27
			ps788.OverlayValues[28] = d28
			ps788.OverlayValues[29] = d29
			ps788.OverlayValues[30] = d30
			ps788.OverlayValues[31] = d31
			ps788.OverlayValues[32] = d32
			ps788.OverlayValues[33] = d33
			ps788.OverlayValues[34] = d34
			ps788.OverlayValues[35] = d35
			ps788.OverlayValues[36] = d36
			ps788.OverlayValues[37] = d37
			ps788.OverlayValues[38] = d38
			ps788.OverlayValues[39] = d39
			ps788.OverlayValues[40] = d40
			ps788.OverlayValues[41] = d41
			ps788.OverlayValues[42] = d42
			ps788.OverlayValues[43] = d43
			ps788.OverlayValues[44] = d44
			ps788.OverlayValues[45] = d45
			ps788.OverlayValues[46] = d46
			ps788.OverlayValues[47] = d47
			ps788.OverlayValues[48] = d48
			ps788.OverlayValues[49] = d49
			ps788.OverlayValues[50] = d50
			ps788.OverlayValues[51] = d51
			ps788.OverlayValues[52] = d52
			ps788.OverlayValues[53] = d53
			ps788.OverlayValues[54] = d54
			ps788.OverlayValues[55] = d55
			ps788.OverlayValues[56] = d56
			ps788.OverlayValues[57] = d57
			ps788.OverlayValues[58] = d58
			ps788.OverlayValues[59] = d59
			ps788.OverlayValues[60] = d60
			ps788.OverlayValues[61] = d61
			ps788.OverlayValues[62] = d62
			ps788.OverlayValues[63] = d63
			ps788.OverlayValues[64] = d64
			ps788.OverlayValues[65] = d65
			ps788.OverlayValues[66] = d66
			ps788.OverlayValues[67] = d67
			ps788.OverlayValues[68] = d68
			ps788.OverlayValues[69] = d69
			ps788.OverlayValues[70] = d70
			ps788.OverlayValues[71] = d71
			ps788.OverlayValues[72] = d72
			ps788.OverlayValues[73] = d73
			ps788.OverlayValues[74] = d74
			ps788.OverlayValues[75] = d75
			ps788.OverlayValues[76] = d76
			ps788.OverlayValues[77] = d77
			ps788.OverlayValues[78] = d78
			ps788.OverlayValues[79] = d79
			ps788.OverlayValues[80] = d80
			ps788.OverlayValues[81] = d81
			ps788.OverlayValues[82] = d82
			ps788.OverlayValues[83] = d83
			ps788.OverlayValues[84] = d84
			ps788.OverlayValues[85] = d85
			ps788.OverlayValues[86] = d86
			ps788.OverlayValues[87] = d87
			ps788.OverlayValues[88] = d88
			ps788.OverlayValues[89] = d89
			ps788.OverlayValues[90] = d90
			ps788.OverlayValues[91] = d91
			ps788.OverlayValues[92] = d92
			ps788.OverlayValues[93] = d93
			ps788.OverlayValues[94] = d94
			ps788.OverlayValues[95] = d95
			ps788.OverlayValues[96] = d96
			ps788.OverlayValues[97] = d97
			ps788.OverlayValues[98] = d98
			ps788.OverlayValues[99] = d99
			ps788.OverlayValues[100] = d100
			ps788.OverlayValues[101] = d101
			ps788.OverlayValues[102] = d102
			ps788.OverlayValues[103] = d103
			ps788.OverlayValues[104] = d104
			ps788.OverlayValues[105] = d105
			ps788.OverlayValues[106] = d106
			ps788.OverlayValues[107] = d107
			ps788.OverlayValues[108] = d108
			ps788.OverlayValues[109] = d109
			ps788.OverlayValues[110] = d110
			ps788.OverlayValues[111] = d111
			ps788.OverlayValues[112] = d112
			ps788.OverlayValues[113] = d113
			ps788.OverlayValues[114] = d114
			ps788.OverlayValues[115] = d115
			ps788.OverlayValues[116] = d116
			ps788.OverlayValues[117] = d117
			ps788.OverlayValues[118] = d118
			ps788.OverlayValues[119] = d119
			ps788.OverlayValues[120] = d120
			ps788.OverlayValues[121] = d121
			ps788.OverlayValues[122] = d122
			ps788.OverlayValues[123] = d123
			ps788.OverlayValues[124] = d124
			ps788.OverlayValues[125] = d125
			ps788.OverlayValues[126] = d126
			ps788.OverlayValues[127] = d127
			ps788.OverlayValues[128] = d128
			ps788.OverlayValues[129] = d129
			ps788.OverlayValues[130] = d130
			ps788.OverlayValues[131] = d131
			ps788.OverlayValues[132] = d132
			ps788.OverlayValues[133] = d133
			ps788.OverlayValues[134] = d134
			ps788.OverlayValues[135] = d135
			ps788.OverlayValues[136] = d136
			ps788.OverlayValues[137] = d137
			ps788.OverlayValues[138] = d138
			ps788.OverlayValues[139] = d139
			ps788.OverlayValues[140] = d140
			ps788.OverlayValues[141] = d141
			ps788.OverlayValues[142] = d142
			ps788.OverlayValues[143] = d143
			ps788.OverlayValues[144] = d144
			ps788.OverlayValues[145] = d145
			ps788.OverlayValues[146] = d146
			ps788.OverlayValues[147] = d147
			ps788.OverlayValues[148] = d148
			ps788.OverlayValues[149] = d149
			ps788.OverlayValues[150] = d150
			ps788.OverlayValues[151] = d151
			ps788.OverlayValues[152] = d152
			ps788.OverlayValues[153] = d153
			ps788.OverlayValues[154] = d154
			ps788.OverlayValues[155] = d155
			ps788.OverlayValues[156] = d156
			ps788.OverlayValues[157] = d157
			ps788.OverlayValues[158] = d158
			ps788.OverlayValues[159] = d159
			ps788.OverlayValues[160] = d160
			ps788.OverlayValues[161] = d161
			ps788.OverlayValues[162] = d162
			ps788.OverlayValues[163] = d163
			ps788.OverlayValues[164] = d164
			ps788.OverlayValues[165] = d165
			ps788.OverlayValues[166] = d166
			ps788.OverlayValues[167] = d167
			ps788.OverlayValues[168] = d168
			ps788.OverlayValues[169] = d169
			ps788.OverlayValues[170] = d170
			ps788.OverlayValues[171] = d171
			ps788.OverlayValues[172] = d172
			ps788.OverlayValues[173] = d173
			ps788.OverlayValues[174] = d174
			ps788.OverlayValues[175] = d175
			ps788.OverlayValues[176] = d176
			ps788.OverlayValues[177] = d177
			ps788.OverlayValues[178] = d178
			ps788.OverlayValues[179] = d179
			ps788.OverlayValues[180] = d180
			ps788.OverlayValues[181] = d181
			ps788.OverlayValues[182] = d182
			ps788.OverlayValues[183] = d183
			ps788.OverlayValues[184] = d184
			ps788.OverlayValues[185] = d185
			ps788.OverlayValues[186] = d186
			ps788.OverlayValues[187] = d187
			ps788.OverlayValues[188] = d188
			ps788.OverlayValues[189] = d189
			ps788.OverlayValues[190] = d190
			ps788.OverlayValues[191] = d191
			ps788.OverlayValues[192] = d192
			ps788.OverlayValues[193] = d193
			ps788.OverlayValues[194] = d194
			ps788.OverlayValues[195] = d195
			ps788.OverlayValues[196] = d196
			ps788.OverlayValues[197] = d197
			ps788.OverlayValues[198] = d198
			ps788.OverlayValues[199] = d199
			ps788.OverlayValues[200] = d200
			ps788.OverlayValues[201] = d201
			ps788.OverlayValues[202] = d202
			ps788.OverlayValues[203] = d203
			ps788.OverlayValues[204] = d204
			ps788.OverlayValues[205] = d205
			ps788.OverlayValues[206] = d206
			ps788.OverlayValues[207] = d207
			ps788.OverlayValues[208] = d208
			ps788.OverlayValues[209] = d209
			ps788.OverlayValues[210] = d210
			ps788.OverlayValues[211] = d211
			ps788.OverlayValues[212] = d212
			ps788.OverlayValues[213] = d213
			ps788.OverlayValues[214] = d214
			ps788.OverlayValues[215] = d215
			ps788.OverlayValues[216] = d216
			ps788.OverlayValues[217] = d217
			ps788.OverlayValues[218] = d218
			ps788.OverlayValues[219] = d219
			ps788.OverlayValues[220] = d220
			ps788.OverlayValues[221] = d221
			ps788.OverlayValues[222] = d222
			ps788.OverlayValues[223] = d223
			ps788.OverlayValues[224] = d224
			ps788.OverlayValues[225] = d225
			ps788.OverlayValues[226] = d226
			ps788.OverlayValues[227] = d227
			ps788.OverlayValues[228] = d228
			ps788.OverlayValues[229] = d229
			ps788.OverlayValues[230] = d230
			ps788.OverlayValues[231] = d231
			ps788.OverlayValues[232] = d232
			ps788.OverlayValues[233] = d233
			ps788.OverlayValues[234] = d234
			ps788.OverlayValues[235] = d235
			ps788.OverlayValues[236] = d236
			ps788.OverlayValues[237] = d237
			ps788.OverlayValues[238] = d238
			ps788.OverlayValues[239] = d239
			ps788.OverlayValues[240] = d240
			ps788.OverlayValues[487] = d487
			ps788.OverlayValues[488] = d488
			ps788.OverlayValues[489] = d489
			ps788.OverlayValues[490] = d490
			ps788.OverlayValues[741] = d741
			ps788.OverlayValues[742] = d742
			ps788.OverlayValues[743] = d743
			ps788.OverlayValues[744] = d744
			ps788.OverlayValues[745] = d745
			ps788.OverlayValues[746] = d746
			ps788.OverlayValues[747] = d747
			ps788.OverlayValues[748] = d748
			ps788.OverlayValues[749] = d749
			ps788.OverlayValues[750] = d750
			ps788.OverlayValues[751] = d751
			ps788.OverlayValues[752] = d752
			ps788.OverlayValues[753] = d753
			ps788.OverlayValues[754] = d754
			ps788.OverlayValues[755] = d755
			ps788.OverlayValues[756] = d756
			ps788.OverlayValues[757] = d757
			ps788.OverlayValues[758] = d758
			ps788.OverlayValues[759] = d759
			ps788.OverlayValues[760] = d760
			ps788.OverlayValues[761] = d761
			ps788.OverlayValues[762] = d762
			ps788.OverlayValues[763] = d763
			ps788.OverlayValues[764] = d764
			ps788.OverlayValues[765] = d765
			ps788.OverlayValues[766] = d766
			ps788.OverlayValues[767] = d767
			ps788.OverlayValues[768] = d768
			ps788.OverlayValues[769] = d769
			ps788.OverlayValues[770] = d770
			ps788.OverlayValues[771] = d771
			ps788.OverlayValues[772] = d772
			ps788.OverlayValues[773] = d773
			ps788.OverlayValues[774] = d774
			ps788.OverlayValues[775] = d775
			ps788.OverlayValues[776] = d776
			ps788.OverlayValues[777] = d777
			ps788.OverlayValues[778] = d778
			ps788.OverlayValues[779] = d779
			ps788.OverlayValues[780] = d780
			ps788.OverlayValues[781] = d781
			ps788.OverlayValues[782] = d782
			ps788.OverlayValues[783] = d783
			ps788.OverlayValues[784] = d784
			ps788.OverlayValues[785] = d785
			ps788.OverlayValues[786] = d786
			ps788.OverlayValues[787] = d787
					return bbs[5].RenderPS(ps788)
				}
			ps789 := scm.PhiState{General: ps.General}
			ps789.OverlayValues = make([]scm.JITValueDesc, 788)
			ps789.OverlayValues[0] = d0
			ps789.OverlayValues[1] = d1
			ps789.OverlayValues[2] = d2
			ps789.OverlayValues[3] = d3
			ps789.OverlayValues[4] = d4
			ps789.OverlayValues[5] = d5
			ps789.OverlayValues[6] = d6
			ps789.OverlayValues[7] = d7
			ps789.OverlayValues[8] = d8
			ps789.OverlayValues[9] = d9
			ps789.OverlayValues[10] = d10
			ps789.OverlayValues[11] = d11
			ps789.OverlayValues[12] = d12
			ps789.OverlayValues[13] = d13
			ps789.OverlayValues[14] = d14
			ps789.OverlayValues[15] = d15
			ps789.OverlayValues[16] = d16
			ps789.OverlayValues[17] = d17
			ps789.OverlayValues[18] = d18
			ps789.OverlayValues[19] = d19
			ps789.OverlayValues[20] = d20
			ps789.OverlayValues[21] = d21
			ps789.OverlayValues[22] = d22
			ps789.OverlayValues[23] = d23
			ps789.OverlayValues[24] = d24
			ps789.OverlayValues[25] = d25
			ps789.OverlayValues[26] = d26
			ps789.OverlayValues[27] = d27
			ps789.OverlayValues[28] = d28
			ps789.OverlayValues[29] = d29
			ps789.OverlayValues[30] = d30
			ps789.OverlayValues[31] = d31
			ps789.OverlayValues[32] = d32
			ps789.OverlayValues[33] = d33
			ps789.OverlayValues[34] = d34
			ps789.OverlayValues[35] = d35
			ps789.OverlayValues[36] = d36
			ps789.OverlayValues[37] = d37
			ps789.OverlayValues[38] = d38
			ps789.OverlayValues[39] = d39
			ps789.OverlayValues[40] = d40
			ps789.OverlayValues[41] = d41
			ps789.OverlayValues[42] = d42
			ps789.OverlayValues[43] = d43
			ps789.OverlayValues[44] = d44
			ps789.OverlayValues[45] = d45
			ps789.OverlayValues[46] = d46
			ps789.OverlayValues[47] = d47
			ps789.OverlayValues[48] = d48
			ps789.OverlayValues[49] = d49
			ps789.OverlayValues[50] = d50
			ps789.OverlayValues[51] = d51
			ps789.OverlayValues[52] = d52
			ps789.OverlayValues[53] = d53
			ps789.OverlayValues[54] = d54
			ps789.OverlayValues[55] = d55
			ps789.OverlayValues[56] = d56
			ps789.OverlayValues[57] = d57
			ps789.OverlayValues[58] = d58
			ps789.OverlayValues[59] = d59
			ps789.OverlayValues[60] = d60
			ps789.OverlayValues[61] = d61
			ps789.OverlayValues[62] = d62
			ps789.OverlayValues[63] = d63
			ps789.OverlayValues[64] = d64
			ps789.OverlayValues[65] = d65
			ps789.OverlayValues[66] = d66
			ps789.OverlayValues[67] = d67
			ps789.OverlayValues[68] = d68
			ps789.OverlayValues[69] = d69
			ps789.OverlayValues[70] = d70
			ps789.OverlayValues[71] = d71
			ps789.OverlayValues[72] = d72
			ps789.OverlayValues[73] = d73
			ps789.OverlayValues[74] = d74
			ps789.OverlayValues[75] = d75
			ps789.OverlayValues[76] = d76
			ps789.OverlayValues[77] = d77
			ps789.OverlayValues[78] = d78
			ps789.OverlayValues[79] = d79
			ps789.OverlayValues[80] = d80
			ps789.OverlayValues[81] = d81
			ps789.OverlayValues[82] = d82
			ps789.OverlayValues[83] = d83
			ps789.OverlayValues[84] = d84
			ps789.OverlayValues[85] = d85
			ps789.OverlayValues[86] = d86
			ps789.OverlayValues[87] = d87
			ps789.OverlayValues[88] = d88
			ps789.OverlayValues[89] = d89
			ps789.OverlayValues[90] = d90
			ps789.OverlayValues[91] = d91
			ps789.OverlayValues[92] = d92
			ps789.OverlayValues[93] = d93
			ps789.OverlayValues[94] = d94
			ps789.OverlayValues[95] = d95
			ps789.OverlayValues[96] = d96
			ps789.OverlayValues[97] = d97
			ps789.OverlayValues[98] = d98
			ps789.OverlayValues[99] = d99
			ps789.OverlayValues[100] = d100
			ps789.OverlayValues[101] = d101
			ps789.OverlayValues[102] = d102
			ps789.OverlayValues[103] = d103
			ps789.OverlayValues[104] = d104
			ps789.OverlayValues[105] = d105
			ps789.OverlayValues[106] = d106
			ps789.OverlayValues[107] = d107
			ps789.OverlayValues[108] = d108
			ps789.OverlayValues[109] = d109
			ps789.OverlayValues[110] = d110
			ps789.OverlayValues[111] = d111
			ps789.OverlayValues[112] = d112
			ps789.OverlayValues[113] = d113
			ps789.OverlayValues[114] = d114
			ps789.OverlayValues[115] = d115
			ps789.OverlayValues[116] = d116
			ps789.OverlayValues[117] = d117
			ps789.OverlayValues[118] = d118
			ps789.OverlayValues[119] = d119
			ps789.OverlayValues[120] = d120
			ps789.OverlayValues[121] = d121
			ps789.OverlayValues[122] = d122
			ps789.OverlayValues[123] = d123
			ps789.OverlayValues[124] = d124
			ps789.OverlayValues[125] = d125
			ps789.OverlayValues[126] = d126
			ps789.OverlayValues[127] = d127
			ps789.OverlayValues[128] = d128
			ps789.OverlayValues[129] = d129
			ps789.OverlayValues[130] = d130
			ps789.OverlayValues[131] = d131
			ps789.OverlayValues[132] = d132
			ps789.OverlayValues[133] = d133
			ps789.OverlayValues[134] = d134
			ps789.OverlayValues[135] = d135
			ps789.OverlayValues[136] = d136
			ps789.OverlayValues[137] = d137
			ps789.OverlayValues[138] = d138
			ps789.OverlayValues[139] = d139
			ps789.OverlayValues[140] = d140
			ps789.OverlayValues[141] = d141
			ps789.OverlayValues[142] = d142
			ps789.OverlayValues[143] = d143
			ps789.OverlayValues[144] = d144
			ps789.OverlayValues[145] = d145
			ps789.OverlayValues[146] = d146
			ps789.OverlayValues[147] = d147
			ps789.OverlayValues[148] = d148
			ps789.OverlayValues[149] = d149
			ps789.OverlayValues[150] = d150
			ps789.OverlayValues[151] = d151
			ps789.OverlayValues[152] = d152
			ps789.OverlayValues[153] = d153
			ps789.OverlayValues[154] = d154
			ps789.OverlayValues[155] = d155
			ps789.OverlayValues[156] = d156
			ps789.OverlayValues[157] = d157
			ps789.OverlayValues[158] = d158
			ps789.OverlayValues[159] = d159
			ps789.OverlayValues[160] = d160
			ps789.OverlayValues[161] = d161
			ps789.OverlayValues[162] = d162
			ps789.OverlayValues[163] = d163
			ps789.OverlayValues[164] = d164
			ps789.OverlayValues[165] = d165
			ps789.OverlayValues[166] = d166
			ps789.OverlayValues[167] = d167
			ps789.OverlayValues[168] = d168
			ps789.OverlayValues[169] = d169
			ps789.OverlayValues[170] = d170
			ps789.OverlayValues[171] = d171
			ps789.OverlayValues[172] = d172
			ps789.OverlayValues[173] = d173
			ps789.OverlayValues[174] = d174
			ps789.OverlayValues[175] = d175
			ps789.OverlayValues[176] = d176
			ps789.OverlayValues[177] = d177
			ps789.OverlayValues[178] = d178
			ps789.OverlayValues[179] = d179
			ps789.OverlayValues[180] = d180
			ps789.OverlayValues[181] = d181
			ps789.OverlayValues[182] = d182
			ps789.OverlayValues[183] = d183
			ps789.OverlayValues[184] = d184
			ps789.OverlayValues[185] = d185
			ps789.OverlayValues[186] = d186
			ps789.OverlayValues[187] = d187
			ps789.OverlayValues[188] = d188
			ps789.OverlayValues[189] = d189
			ps789.OverlayValues[190] = d190
			ps789.OverlayValues[191] = d191
			ps789.OverlayValues[192] = d192
			ps789.OverlayValues[193] = d193
			ps789.OverlayValues[194] = d194
			ps789.OverlayValues[195] = d195
			ps789.OverlayValues[196] = d196
			ps789.OverlayValues[197] = d197
			ps789.OverlayValues[198] = d198
			ps789.OverlayValues[199] = d199
			ps789.OverlayValues[200] = d200
			ps789.OverlayValues[201] = d201
			ps789.OverlayValues[202] = d202
			ps789.OverlayValues[203] = d203
			ps789.OverlayValues[204] = d204
			ps789.OverlayValues[205] = d205
			ps789.OverlayValues[206] = d206
			ps789.OverlayValues[207] = d207
			ps789.OverlayValues[208] = d208
			ps789.OverlayValues[209] = d209
			ps789.OverlayValues[210] = d210
			ps789.OverlayValues[211] = d211
			ps789.OverlayValues[212] = d212
			ps789.OverlayValues[213] = d213
			ps789.OverlayValues[214] = d214
			ps789.OverlayValues[215] = d215
			ps789.OverlayValues[216] = d216
			ps789.OverlayValues[217] = d217
			ps789.OverlayValues[218] = d218
			ps789.OverlayValues[219] = d219
			ps789.OverlayValues[220] = d220
			ps789.OverlayValues[221] = d221
			ps789.OverlayValues[222] = d222
			ps789.OverlayValues[223] = d223
			ps789.OverlayValues[224] = d224
			ps789.OverlayValues[225] = d225
			ps789.OverlayValues[226] = d226
			ps789.OverlayValues[227] = d227
			ps789.OverlayValues[228] = d228
			ps789.OverlayValues[229] = d229
			ps789.OverlayValues[230] = d230
			ps789.OverlayValues[231] = d231
			ps789.OverlayValues[232] = d232
			ps789.OverlayValues[233] = d233
			ps789.OverlayValues[234] = d234
			ps789.OverlayValues[235] = d235
			ps789.OverlayValues[236] = d236
			ps789.OverlayValues[237] = d237
			ps789.OverlayValues[238] = d238
			ps789.OverlayValues[239] = d239
			ps789.OverlayValues[240] = d240
			ps789.OverlayValues[487] = d487
			ps789.OverlayValues[488] = d488
			ps789.OverlayValues[489] = d489
			ps789.OverlayValues[490] = d490
			ps789.OverlayValues[741] = d741
			ps789.OverlayValues[742] = d742
			ps789.OverlayValues[743] = d743
			ps789.OverlayValues[744] = d744
			ps789.OverlayValues[745] = d745
			ps789.OverlayValues[746] = d746
			ps789.OverlayValues[747] = d747
			ps789.OverlayValues[748] = d748
			ps789.OverlayValues[749] = d749
			ps789.OverlayValues[750] = d750
			ps789.OverlayValues[751] = d751
			ps789.OverlayValues[752] = d752
			ps789.OverlayValues[753] = d753
			ps789.OverlayValues[754] = d754
			ps789.OverlayValues[755] = d755
			ps789.OverlayValues[756] = d756
			ps789.OverlayValues[757] = d757
			ps789.OverlayValues[758] = d758
			ps789.OverlayValues[759] = d759
			ps789.OverlayValues[760] = d760
			ps789.OverlayValues[761] = d761
			ps789.OverlayValues[762] = d762
			ps789.OverlayValues[763] = d763
			ps789.OverlayValues[764] = d764
			ps789.OverlayValues[765] = d765
			ps789.OverlayValues[766] = d766
			ps789.OverlayValues[767] = d767
			ps789.OverlayValues[768] = d768
			ps789.OverlayValues[769] = d769
			ps789.OverlayValues[770] = d770
			ps789.OverlayValues[771] = d771
			ps789.OverlayValues[772] = d772
			ps789.OverlayValues[773] = d773
			ps789.OverlayValues[774] = d774
			ps789.OverlayValues[775] = d775
			ps789.OverlayValues[776] = d776
			ps789.OverlayValues[777] = d777
			ps789.OverlayValues[778] = d778
			ps789.OverlayValues[779] = d779
			ps789.OverlayValues[780] = d780
			ps789.OverlayValues[781] = d781
			ps789.OverlayValues[782] = d782
			ps789.OverlayValues[783] = d783
			ps789.OverlayValues[784] = d784
			ps789.OverlayValues[785] = d785
			ps789.OverlayValues[786] = d786
			ps789.OverlayValues[787] = d787
				return bbs[7].RenderPS(ps789)
			}
			if !ps.General {
				ps.General = true
				return bbs[4].RenderPS(ps)
			}
			lbl80 := ctx.ReserveLabel()
			lbl81 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d787.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl80)
			ctx.EmitJmp(lbl81)
			ctx.MarkLabel(lbl80)
			ctx.EmitJmp(lbl6)
			ctx.MarkLabel(lbl81)
			ctx.EmitJmp(lbl8)
			ps790 := scm.PhiState{General: true}
			ps790.OverlayValues = make([]scm.JITValueDesc, 788)
			ps790.OverlayValues[0] = d0
			ps790.OverlayValues[1] = d1
			ps790.OverlayValues[2] = d2
			ps790.OverlayValues[3] = d3
			ps790.OverlayValues[4] = d4
			ps790.OverlayValues[5] = d5
			ps790.OverlayValues[6] = d6
			ps790.OverlayValues[7] = d7
			ps790.OverlayValues[8] = d8
			ps790.OverlayValues[9] = d9
			ps790.OverlayValues[10] = d10
			ps790.OverlayValues[11] = d11
			ps790.OverlayValues[12] = d12
			ps790.OverlayValues[13] = d13
			ps790.OverlayValues[14] = d14
			ps790.OverlayValues[15] = d15
			ps790.OverlayValues[16] = d16
			ps790.OverlayValues[17] = d17
			ps790.OverlayValues[18] = d18
			ps790.OverlayValues[19] = d19
			ps790.OverlayValues[20] = d20
			ps790.OverlayValues[21] = d21
			ps790.OverlayValues[22] = d22
			ps790.OverlayValues[23] = d23
			ps790.OverlayValues[24] = d24
			ps790.OverlayValues[25] = d25
			ps790.OverlayValues[26] = d26
			ps790.OverlayValues[27] = d27
			ps790.OverlayValues[28] = d28
			ps790.OverlayValues[29] = d29
			ps790.OverlayValues[30] = d30
			ps790.OverlayValues[31] = d31
			ps790.OverlayValues[32] = d32
			ps790.OverlayValues[33] = d33
			ps790.OverlayValues[34] = d34
			ps790.OverlayValues[35] = d35
			ps790.OverlayValues[36] = d36
			ps790.OverlayValues[37] = d37
			ps790.OverlayValues[38] = d38
			ps790.OverlayValues[39] = d39
			ps790.OverlayValues[40] = d40
			ps790.OverlayValues[41] = d41
			ps790.OverlayValues[42] = d42
			ps790.OverlayValues[43] = d43
			ps790.OverlayValues[44] = d44
			ps790.OverlayValues[45] = d45
			ps790.OverlayValues[46] = d46
			ps790.OverlayValues[47] = d47
			ps790.OverlayValues[48] = d48
			ps790.OverlayValues[49] = d49
			ps790.OverlayValues[50] = d50
			ps790.OverlayValues[51] = d51
			ps790.OverlayValues[52] = d52
			ps790.OverlayValues[53] = d53
			ps790.OverlayValues[54] = d54
			ps790.OverlayValues[55] = d55
			ps790.OverlayValues[56] = d56
			ps790.OverlayValues[57] = d57
			ps790.OverlayValues[58] = d58
			ps790.OverlayValues[59] = d59
			ps790.OverlayValues[60] = d60
			ps790.OverlayValues[61] = d61
			ps790.OverlayValues[62] = d62
			ps790.OverlayValues[63] = d63
			ps790.OverlayValues[64] = d64
			ps790.OverlayValues[65] = d65
			ps790.OverlayValues[66] = d66
			ps790.OverlayValues[67] = d67
			ps790.OverlayValues[68] = d68
			ps790.OverlayValues[69] = d69
			ps790.OverlayValues[70] = d70
			ps790.OverlayValues[71] = d71
			ps790.OverlayValues[72] = d72
			ps790.OverlayValues[73] = d73
			ps790.OverlayValues[74] = d74
			ps790.OverlayValues[75] = d75
			ps790.OverlayValues[76] = d76
			ps790.OverlayValues[77] = d77
			ps790.OverlayValues[78] = d78
			ps790.OverlayValues[79] = d79
			ps790.OverlayValues[80] = d80
			ps790.OverlayValues[81] = d81
			ps790.OverlayValues[82] = d82
			ps790.OverlayValues[83] = d83
			ps790.OverlayValues[84] = d84
			ps790.OverlayValues[85] = d85
			ps790.OverlayValues[86] = d86
			ps790.OverlayValues[87] = d87
			ps790.OverlayValues[88] = d88
			ps790.OverlayValues[89] = d89
			ps790.OverlayValues[90] = d90
			ps790.OverlayValues[91] = d91
			ps790.OverlayValues[92] = d92
			ps790.OverlayValues[93] = d93
			ps790.OverlayValues[94] = d94
			ps790.OverlayValues[95] = d95
			ps790.OverlayValues[96] = d96
			ps790.OverlayValues[97] = d97
			ps790.OverlayValues[98] = d98
			ps790.OverlayValues[99] = d99
			ps790.OverlayValues[100] = d100
			ps790.OverlayValues[101] = d101
			ps790.OverlayValues[102] = d102
			ps790.OverlayValues[103] = d103
			ps790.OverlayValues[104] = d104
			ps790.OverlayValues[105] = d105
			ps790.OverlayValues[106] = d106
			ps790.OverlayValues[107] = d107
			ps790.OverlayValues[108] = d108
			ps790.OverlayValues[109] = d109
			ps790.OverlayValues[110] = d110
			ps790.OverlayValues[111] = d111
			ps790.OverlayValues[112] = d112
			ps790.OverlayValues[113] = d113
			ps790.OverlayValues[114] = d114
			ps790.OverlayValues[115] = d115
			ps790.OverlayValues[116] = d116
			ps790.OverlayValues[117] = d117
			ps790.OverlayValues[118] = d118
			ps790.OverlayValues[119] = d119
			ps790.OverlayValues[120] = d120
			ps790.OverlayValues[121] = d121
			ps790.OverlayValues[122] = d122
			ps790.OverlayValues[123] = d123
			ps790.OverlayValues[124] = d124
			ps790.OverlayValues[125] = d125
			ps790.OverlayValues[126] = d126
			ps790.OverlayValues[127] = d127
			ps790.OverlayValues[128] = d128
			ps790.OverlayValues[129] = d129
			ps790.OverlayValues[130] = d130
			ps790.OverlayValues[131] = d131
			ps790.OverlayValues[132] = d132
			ps790.OverlayValues[133] = d133
			ps790.OverlayValues[134] = d134
			ps790.OverlayValues[135] = d135
			ps790.OverlayValues[136] = d136
			ps790.OverlayValues[137] = d137
			ps790.OverlayValues[138] = d138
			ps790.OverlayValues[139] = d139
			ps790.OverlayValues[140] = d140
			ps790.OverlayValues[141] = d141
			ps790.OverlayValues[142] = d142
			ps790.OverlayValues[143] = d143
			ps790.OverlayValues[144] = d144
			ps790.OverlayValues[145] = d145
			ps790.OverlayValues[146] = d146
			ps790.OverlayValues[147] = d147
			ps790.OverlayValues[148] = d148
			ps790.OverlayValues[149] = d149
			ps790.OverlayValues[150] = d150
			ps790.OverlayValues[151] = d151
			ps790.OverlayValues[152] = d152
			ps790.OverlayValues[153] = d153
			ps790.OverlayValues[154] = d154
			ps790.OverlayValues[155] = d155
			ps790.OverlayValues[156] = d156
			ps790.OverlayValues[157] = d157
			ps790.OverlayValues[158] = d158
			ps790.OverlayValues[159] = d159
			ps790.OverlayValues[160] = d160
			ps790.OverlayValues[161] = d161
			ps790.OverlayValues[162] = d162
			ps790.OverlayValues[163] = d163
			ps790.OverlayValues[164] = d164
			ps790.OverlayValues[165] = d165
			ps790.OverlayValues[166] = d166
			ps790.OverlayValues[167] = d167
			ps790.OverlayValues[168] = d168
			ps790.OverlayValues[169] = d169
			ps790.OverlayValues[170] = d170
			ps790.OverlayValues[171] = d171
			ps790.OverlayValues[172] = d172
			ps790.OverlayValues[173] = d173
			ps790.OverlayValues[174] = d174
			ps790.OverlayValues[175] = d175
			ps790.OverlayValues[176] = d176
			ps790.OverlayValues[177] = d177
			ps790.OverlayValues[178] = d178
			ps790.OverlayValues[179] = d179
			ps790.OverlayValues[180] = d180
			ps790.OverlayValues[181] = d181
			ps790.OverlayValues[182] = d182
			ps790.OverlayValues[183] = d183
			ps790.OverlayValues[184] = d184
			ps790.OverlayValues[185] = d185
			ps790.OverlayValues[186] = d186
			ps790.OverlayValues[187] = d187
			ps790.OverlayValues[188] = d188
			ps790.OverlayValues[189] = d189
			ps790.OverlayValues[190] = d190
			ps790.OverlayValues[191] = d191
			ps790.OverlayValues[192] = d192
			ps790.OverlayValues[193] = d193
			ps790.OverlayValues[194] = d194
			ps790.OverlayValues[195] = d195
			ps790.OverlayValues[196] = d196
			ps790.OverlayValues[197] = d197
			ps790.OverlayValues[198] = d198
			ps790.OverlayValues[199] = d199
			ps790.OverlayValues[200] = d200
			ps790.OverlayValues[201] = d201
			ps790.OverlayValues[202] = d202
			ps790.OverlayValues[203] = d203
			ps790.OverlayValues[204] = d204
			ps790.OverlayValues[205] = d205
			ps790.OverlayValues[206] = d206
			ps790.OverlayValues[207] = d207
			ps790.OverlayValues[208] = d208
			ps790.OverlayValues[209] = d209
			ps790.OverlayValues[210] = d210
			ps790.OverlayValues[211] = d211
			ps790.OverlayValues[212] = d212
			ps790.OverlayValues[213] = d213
			ps790.OverlayValues[214] = d214
			ps790.OverlayValues[215] = d215
			ps790.OverlayValues[216] = d216
			ps790.OverlayValues[217] = d217
			ps790.OverlayValues[218] = d218
			ps790.OverlayValues[219] = d219
			ps790.OverlayValues[220] = d220
			ps790.OverlayValues[221] = d221
			ps790.OverlayValues[222] = d222
			ps790.OverlayValues[223] = d223
			ps790.OverlayValues[224] = d224
			ps790.OverlayValues[225] = d225
			ps790.OverlayValues[226] = d226
			ps790.OverlayValues[227] = d227
			ps790.OverlayValues[228] = d228
			ps790.OverlayValues[229] = d229
			ps790.OverlayValues[230] = d230
			ps790.OverlayValues[231] = d231
			ps790.OverlayValues[232] = d232
			ps790.OverlayValues[233] = d233
			ps790.OverlayValues[234] = d234
			ps790.OverlayValues[235] = d235
			ps790.OverlayValues[236] = d236
			ps790.OverlayValues[237] = d237
			ps790.OverlayValues[238] = d238
			ps790.OverlayValues[239] = d239
			ps790.OverlayValues[240] = d240
			ps790.OverlayValues[487] = d487
			ps790.OverlayValues[488] = d488
			ps790.OverlayValues[489] = d489
			ps790.OverlayValues[490] = d490
			ps790.OverlayValues[741] = d741
			ps790.OverlayValues[742] = d742
			ps790.OverlayValues[743] = d743
			ps790.OverlayValues[744] = d744
			ps790.OverlayValues[745] = d745
			ps790.OverlayValues[746] = d746
			ps790.OverlayValues[747] = d747
			ps790.OverlayValues[748] = d748
			ps790.OverlayValues[749] = d749
			ps790.OverlayValues[750] = d750
			ps790.OverlayValues[751] = d751
			ps790.OverlayValues[752] = d752
			ps790.OverlayValues[753] = d753
			ps790.OverlayValues[754] = d754
			ps790.OverlayValues[755] = d755
			ps790.OverlayValues[756] = d756
			ps790.OverlayValues[757] = d757
			ps790.OverlayValues[758] = d758
			ps790.OverlayValues[759] = d759
			ps790.OverlayValues[760] = d760
			ps790.OverlayValues[761] = d761
			ps790.OverlayValues[762] = d762
			ps790.OverlayValues[763] = d763
			ps790.OverlayValues[764] = d764
			ps790.OverlayValues[765] = d765
			ps790.OverlayValues[766] = d766
			ps790.OverlayValues[767] = d767
			ps790.OverlayValues[768] = d768
			ps790.OverlayValues[769] = d769
			ps790.OverlayValues[770] = d770
			ps790.OverlayValues[771] = d771
			ps790.OverlayValues[772] = d772
			ps790.OverlayValues[773] = d773
			ps790.OverlayValues[774] = d774
			ps790.OverlayValues[775] = d775
			ps790.OverlayValues[776] = d776
			ps790.OverlayValues[777] = d777
			ps790.OverlayValues[778] = d778
			ps790.OverlayValues[779] = d779
			ps790.OverlayValues[780] = d780
			ps790.OverlayValues[781] = d781
			ps790.OverlayValues[782] = d782
			ps790.OverlayValues[783] = d783
			ps790.OverlayValues[784] = d784
			ps790.OverlayValues[785] = d785
			ps790.OverlayValues[786] = d786
			ps790.OverlayValues[787] = d787
			ps791 := scm.PhiState{General: true}
			ps791.OverlayValues = make([]scm.JITValueDesc, 788)
			ps791.OverlayValues[0] = d0
			ps791.OverlayValues[1] = d1
			ps791.OverlayValues[2] = d2
			ps791.OverlayValues[3] = d3
			ps791.OverlayValues[4] = d4
			ps791.OverlayValues[5] = d5
			ps791.OverlayValues[6] = d6
			ps791.OverlayValues[7] = d7
			ps791.OverlayValues[8] = d8
			ps791.OverlayValues[9] = d9
			ps791.OverlayValues[10] = d10
			ps791.OverlayValues[11] = d11
			ps791.OverlayValues[12] = d12
			ps791.OverlayValues[13] = d13
			ps791.OverlayValues[14] = d14
			ps791.OverlayValues[15] = d15
			ps791.OverlayValues[16] = d16
			ps791.OverlayValues[17] = d17
			ps791.OverlayValues[18] = d18
			ps791.OverlayValues[19] = d19
			ps791.OverlayValues[20] = d20
			ps791.OverlayValues[21] = d21
			ps791.OverlayValues[22] = d22
			ps791.OverlayValues[23] = d23
			ps791.OverlayValues[24] = d24
			ps791.OverlayValues[25] = d25
			ps791.OverlayValues[26] = d26
			ps791.OverlayValues[27] = d27
			ps791.OverlayValues[28] = d28
			ps791.OverlayValues[29] = d29
			ps791.OverlayValues[30] = d30
			ps791.OverlayValues[31] = d31
			ps791.OverlayValues[32] = d32
			ps791.OverlayValues[33] = d33
			ps791.OverlayValues[34] = d34
			ps791.OverlayValues[35] = d35
			ps791.OverlayValues[36] = d36
			ps791.OverlayValues[37] = d37
			ps791.OverlayValues[38] = d38
			ps791.OverlayValues[39] = d39
			ps791.OverlayValues[40] = d40
			ps791.OverlayValues[41] = d41
			ps791.OverlayValues[42] = d42
			ps791.OverlayValues[43] = d43
			ps791.OverlayValues[44] = d44
			ps791.OverlayValues[45] = d45
			ps791.OverlayValues[46] = d46
			ps791.OverlayValues[47] = d47
			ps791.OverlayValues[48] = d48
			ps791.OverlayValues[49] = d49
			ps791.OverlayValues[50] = d50
			ps791.OverlayValues[51] = d51
			ps791.OverlayValues[52] = d52
			ps791.OverlayValues[53] = d53
			ps791.OverlayValues[54] = d54
			ps791.OverlayValues[55] = d55
			ps791.OverlayValues[56] = d56
			ps791.OverlayValues[57] = d57
			ps791.OverlayValues[58] = d58
			ps791.OverlayValues[59] = d59
			ps791.OverlayValues[60] = d60
			ps791.OverlayValues[61] = d61
			ps791.OverlayValues[62] = d62
			ps791.OverlayValues[63] = d63
			ps791.OverlayValues[64] = d64
			ps791.OverlayValues[65] = d65
			ps791.OverlayValues[66] = d66
			ps791.OverlayValues[67] = d67
			ps791.OverlayValues[68] = d68
			ps791.OverlayValues[69] = d69
			ps791.OverlayValues[70] = d70
			ps791.OverlayValues[71] = d71
			ps791.OverlayValues[72] = d72
			ps791.OverlayValues[73] = d73
			ps791.OverlayValues[74] = d74
			ps791.OverlayValues[75] = d75
			ps791.OverlayValues[76] = d76
			ps791.OverlayValues[77] = d77
			ps791.OverlayValues[78] = d78
			ps791.OverlayValues[79] = d79
			ps791.OverlayValues[80] = d80
			ps791.OverlayValues[81] = d81
			ps791.OverlayValues[82] = d82
			ps791.OverlayValues[83] = d83
			ps791.OverlayValues[84] = d84
			ps791.OverlayValues[85] = d85
			ps791.OverlayValues[86] = d86
			ps791.OverlayValues[87] = d87
			ps791.OverlayValues[88] = d88
			ps791.OverlayValues[89] = d89
			ps791.OverlayValues[90] = d90
			ps791.OverlayValues[91] = d91
			ps791.OverlayValues[92] = d92
			ps791.OverlayValues[93] = d93
			ps791.OverlayValues[94] = d94
			ps791.OverlayValues[95] = d95
			ps791.OverlayValues[96] = d96
			ps791.OverlayValues[97] = d97
			ps791.OverlayValues[98] = d98
			ps791.OverlayValues[99] = d99
			ps791.OverlayValues[100] = d100
			ps791.OverlayValues[101] = d101
			ps791.OverlayValues[102] = d102
			ps791.OverlayValues[103] = d103
			ps791.OverlayValues[104] = d104
			ps791.OverlayValues[105] = d105
			ps791.OverlayValues[106] = d106
			ps791.OverlayValues[107] = d107
			ps791.OverlayValues[108] = d108
			ps791.OverlayValues[109] = d109
			ps791.OverlayValues[110] = d110
			ps791.OverlayValues[111] = d111
			ps791.OverlayValues[112] = d112
			ps791.OverlayValues[113] = d113
			ps791.OverlayValues[114] = d114
			ps791.OverlayValues[115] = d115
			ps791.OverlayValues[116] = d116
			ps791.OverlayValues[117] = d117
			ps791.OverlayValues[118] = d118
			ps791.OverlayValues[119] = d119
			ps791.OverlayValues[120] = d120
			ps791.OverlayValues[121] = d121
			ps791.OverlayValues[122] = d122
			ps791.OverlayValues[123] = d123
			ps791.OverlayValues[124] = d124
			ps791.OverlayValues[125] = d125
			ps791.OverlayValues[126] = d126
			ps791.OverlayValues[127] = d127
			ps791.OverlayValues[128] = d128
			ps791.OverlayValues[129] = d129
			ps791.OverlayValues[130] = d130
			ps791.OverlayValues[131] = d131
			ps791.OverlayValues[132] = d132
			ps791.OverlayValues[133] = d133
			ps791.OverlayValues[134] = d134
			ps791.OverlayValues[135] = d135
			ps791.OverlayValues[136] = d136
			ps791.OverlayValues[137] = d137
			ps791.OverlayValues[138] = d138
			ps791.OverlayValues[139] = d139
			ps791.OverlayValues[140] = d140
			ps791.OverlayValues[141] = d141
			ps791.OverlayValues[142] = d142
			ps791.OverlayValues[143] = d143
			ps791.OverlayValues[144] = d144
			ps791.OverlayValues[145] = d145
			ps791.OverlayValues[146] = d146
			ps791.OverlayValues[147] = d147
			ps791.OverlayValues[148] = d148
			ps791.OverlayValues[149] = d149
			ps791.OverlayValues[150] = d150
			ps791.OverlayValues[151] = d151
			ps791.OverlayValues[152] = d152
			ps791.OverlayValues[153] = d153
			ps791.OverlayValues[154] = d154
			ps791.OverlayValues[155] = d155
			ps791.OverlayValues[156] = d156
			ps791.OverlayValues[157] = d157
			ps791.OverlayValues[158] = d158
			ps791.OverlayValues[159] = d159
			ps791.OverlayValues[160] = d160
			ps791.OverlayValues[161] = d161
			ps791.OverlayValues[162] = d162
			ps791.OverlayValues[163] = d163
			ps791.OverlayValues[164] = d164
			ps791.OverlayValues[165] = d165
			ps791.OverlayValues[166] = d166
			ps791.OverlayValues[167] = d167
			ps791.OverlayValues[168] = d168
			ps791.OverlayValues[169] = d169
			ps791.OverlayValues[170] = d170
			ps791.OverlayValues[171] = d171
			ps791.OverlayValues[172] = d172
			ps791.OverlayValues[173] = d173
			ps791.OverlayValues[174] = d174
			ps791.OverlayValues[175] = d175
			ps791.OverlayValues[176] = d176
			ps791.OverlayValues[177] = d177
			ps791.OverlayValues[178] = d178
			ps791.OverlayValues[179] = d179
			ps791.OverlayValues[180] = d180
			ps791.OverlayValues[181] = d181
			ps791.OverlayValues[182] = d182
			ps791.OverlayValues[183] = d183
			ps791.OverlayValues[184] = d184
			ps791.OverlayValues[185] = d185
			ps791.OverlayValues[186] = d186
			ps791.OverlayValues[187] = d187
			ps791.OverlayValues[188] = d188
			ps791.OverlayValues[189] = d189
			ps791.OverlayValues[190] = d190
			ps791.OverlayValues[191] = d191
			ps791.OverlayValues[192] = d192
			ps791.OverlayValues[193] = d193
			ps791.OverlayValues[194] = d194
			ps791.OverlayValues[195] = d195
			ps791.OverlayValues[196] = d196
			ps791.OverlayValues[197] = d197
			ps791.OverlayValues[198] = d198
			ps791.OverlayValues[199] = d199
			ps791.OverlayValues[200] = d200
			ps791.OverlayValues[201] = d201
			ps791.OverlayValues[202] = d202
			ps791.OverlayValues[203] = d203
			ps791.OverlayValues[204] = d204
			ps791.OverlayValues[205] = d205
			ps791.OverlayValues[206] = d206
			ps791.OverlayValues[207] = d207
			ps791.OverlayValues[208] = d208
			ps791.OverlayValues[209] = d209
			ps791.OverlayValues[210] = d210
			ps791.OverlayValues[211] = d211
			ps791.OverlayValues[212] = d212
			ps791.OverlayValues[213] = d213
			ps791.OverlayValues[214] = d214
			ps791.OverlayValues[215] = d215
			ps791.OverlayValues[216] = d216
			ps791.OverlayValues[217] = d217
			ps791.OverlayValues[218] = d218
			ps791.OverlayValues[219] = d219
			ps791.OverlayValues[220] = d220
			ps791.OverlayValues[221] = d221
			ps791.OverlayValues[222] = d222
			ps791.OverlayValues[223] = d223
			ps791.OverlayValues[224] = d224
			ps791.OverlayValues[225] = d225
			ps791.OverlayValues[226] = d226
			ps791.OverlayValues[227] = d227
			ps791.OverlayValues[228] = d228
			ps791.OverlayValues[229] = d229
			ps791.OverlayValues[230] = d230
			ps791.OverlayValues[231] = d231
			ps791.OverlayValues[232] = d232
			ps791.OverlayValues[233] = d233
			ps791.OverlayValues[234] = d234
			ps791.OverlayValues[235] = d235
			ps791.OverlayValues[236] = d236
			ps791.OverlayValues[237] = d237
			ps791.OverlayValues[238] = d238
			ps791.OverlayValues[239] = d239
			ps791.OverlayValues[240] = d240
			ps791.OverlayValues[487] = d487
			ps791.OverlayValues[488] = d488
			ps791.OverlayValues[489] = d489
			ps791.OverlayValues[490] = d490
			ps791.OverlayValues[741] = d741
			ps791.OverlayValues[742] = d742
			ps791.OverlayValues[743] = d743
			ps791.OverlayValues[744] = d744
			ps791.OverlayValues[745] = d745
			ps791.OverlayValues[746] = d746
			ps791.OverlayValues[747] = d747
			ps791.OverlayValues[748] = d748
			ps791.OverlayValues[749] = d749
			ps791.OverlayValues[750] = d750
			ps791.OverlayValues[751] = d751
			ps791.OverlayValues[752] = d752
			ps791.OverlayValues[753] = d753
			ps791.OverlayValues[754] = d754
			ps791.OverlayValues[755] = d755
			ps791.OverlayValues[756] = d756
			ps791.OverlayValues[757] = d757
			ps791.OverlayValues[758] = d758
			ps791.OverlayValues[759] = d759
			ps791.OverlayValues[760] = d760
			ps791.OverlayValues[761] = d761
			ps791.OverlayValues[762] = d762
			ps791.OverlayValues[763] = d763
			ps791.OverlayValues[764] = d764
			ps791.OverlayValues[765] = d765
			ps791.OverlayValues[766] = d766
			ps791.OverlayValues[767] = d767
			ps791.OverlayValues[768] = d768
			ps791.OverlayValues[769] = d769
			ps791.OverlayValues[770] = d770
			ps791.OverlayValues[771] = d771
			ps791.OverlayValues[772] = d772
			ps791.OverlayValues[773] = d773
			ps791.OverlayValues[774] = d774
			ps791.OverlayValues[775] = d775
			ps791.OverlayValues[776] = d776
			ps791.OverlayValues[777] = d777
			ps791.OverlayValues[778] = d778
			ps791.OverlayValues[779] = d779
			ps791.OverlayValues[780] = d780
			ps791.OverlayValues[781] = d781
			ps791.OverlayValues[782] = d782
			ps791.OverlayValues[783] = d783
			ps791.OverlayValues[784] = d784
			ps791.OverlayValues[785] = d785
			ps791.OverlayValues[786] = d786
			ps791.OverlayValues[787] = d787
			snap792 := d0
			snap793 := d1
			snap794 := d2
			snap795 := d3
			snap796 := d4
			snap797 := d5
			snap798 := d6
			snap799 := d7
			snap800 := d8
			snap801 := d9
			snap802 := d10
			snap803 := d11
			snap804 := d12
			snap805 := d13
			snap806 := d14
			snap807 := d15
			snap808 := d16
			snap809 := d17
			snap810 := d18
			snap811 := d19
			snap812 := d20
			snap813 := d21
			snap814 := d22
			snap815 := d23
			snap816 := d24
			snap817 := d25
			snap818 := d26
			snap819 := d27
			snap820 := d28
			snap821 := d29
			snap822 := d30
			snap823 := d31
			snap824 := d32
			snap825 := d33
			snap826 := d34
			snap827 := d35
			snap828 := d36
			snap829 := d37
			snap830 := d38
			snap831 := d39
			snap832 := d40
			snap833 := d41
			snap834 := d42
			snap835 := d43
			snap836 := d44
			snap837 := d45
			snap838 := d46
			snap839 := d47
			snap840 := d48
			snap841 := d49
			snap842 := d50
			snap843 := d51
			snap844 := d52
			snap845 := d53
			snap846 := d54
			snap847 := d55
			snap848 := d56
			snap849 := d57
			snap850 := d58
			snap851 := d59
			snap852 := d60
			snap853 := d61
			snap854 := d62
			snap855 := d63
			snap856 := d64
			snap857 := d65
			snap858 := d66
			snap859 := d67
			snap860 := d68
			snap861 := d69
			snap862 := d70
			snap863 := d71
			snap864 := d72
			snap865 := d73
			snap866 := d74
			snap867 := d75
			snap868 := d76
			snap869 := d77
			snap870 := d78
			snap871 := d79
			snap872 := d80
			snap873 := d81
			snap874 := d82
			snap875 := d83
			snap876 := d84
			snap877 := d85
			snap878 := d86
			snap879 := d87
			snap880 := d88
			snap881 := d89
			snap882 := d90
			snap883 := d91
			snap884 := d92
			snap885 := d93
			snap886 := d94
			snap887 := d95
			snap888 := d96
			snap889 := d97
			snap890 := d98
			snap891 := d99
			snap892 := d100
			snap893 := d101
			snap894 := d102
			snap895 := d103
			snap896 := d104
			snap897 := d105
			snap898 := d106
			snap899 := d107
			snap900 := d108
			snap901 := d109
			snap902 := d110
			snap903 := d111
			snap904 := d112
			snap905 := d113
			snap906 := d114
			snap907 := d115
			snap908 := d116
			snap909 := d117
			snap910 := d118
			snap911 := d119
			snap912 := d120
			snap913 := d121
			snap914 := d122
			snap915 := d123
			snap916 := d124
			snap917 := d125
			snap918 := d126
			snap919 := d127
			snap920 := d128
			snap921 := d129
			snap922 := d130
			snap923 := d131
			snap924 := d132
			snap925 := d133
			snap926 := d134
			snap927 := d135
			snap928 := d136
			snap929 := d137
			snap930 := d138
			snap931 := d139
			snap932 := d140
			snap933 := d141
			snap934 := d142
			snap935 := d143
			snap936 := d144
			snap937 := d145
			snap938 := d146
			snap939 := d147
			snap940 := d148
			snap941 := d149
			snap942 := d150
			snap943 := d151
			snap944 := d152
			snap945 := d153
			snap946 := d154
			snap947 := d155
			snap948 := d156
			snap949 := d157
			snap950 := d158
			snap951 := d159
			snap952 := d160
			snap953 := d161
			snap954 := d162
			snap955 := d163
			snap956 := d164
			snap957 := d165
			snap958 := d166
			snap959 := d167
			snap960 := d168
			snap961 := d169
			snap962 := d170
			snap963 := d171
			snap964 := d172
			snap965 := d173
			snap966 := d174
			snap967 := d175
			snap968 := d176
			snap969 := d177
			snap970 := d178
			snap971 := d179
			snap972 := d180
			snap973 := d181
			snap974 := d182
			snap975 := d183
			snap976 := d184
			snap977 := d185
			snap978 := d186
			snap979 := d187
			snap980 := d188
			snap981 := d189
			snap982 := d190
			snap983 := d191
			snap984 := d192
			snap985 := d193
			snap986 := d194
			snap987 := d195
			snap988 := d196
			snap989 := d197
			snap990 := d198
			snap991 := d199
			snap992 := d200
			snap993 := d201
			snap994 := d202
			snap995 := d203
			snap996 := d204
			snap997 := d205
			snap998 := d206
			snap999 := d207
			snap1000 := d208
			snap1001 := d209
			snap1002 := d210
			snap1003 := d211
			snap1004 := d212
			snap1005 := d213
			snap1006 := d214
			snap1007 := d215
			snap1008 := d216
			snap1009 := d217
			snap1010 := d218
			snap1011 := d219
			snap1012 := d220
			snap1013 := d221
			snap1014 := d222
			snap1015 := d223
			snap1016 := d224
			snap1017 := d225
			snap1018 := d226
			snap1019 := d227
			snap1020 := d228
			snap1021 := d229
			snap1022 := d230
			snap1023 := d231
			snap1024 := d232
			snap1025 := d233
			snap1026 := d234
			snap1027 := d235
			snap1028 := d236
			snap1029 := d237
			snap1030 := d238
			snap1031 := d239
			snap1032 := d240
			snap1033 := d487
			snap1034 := d488
			snap1035 := d489
			snap1036 := d490
			snap1037 := d741
			snap1038 := d742
			snap1039 := d743
			snap1040 := d744
			snap1041 := d745
			snap1042 := d746
			snap1043 := d747
			snap1044 := d748
			snap1045 := d749
			snap1046 := d750
			snap1047 := d751
			snap1048 := d752
			snap1049 := d753
			snap1050 := d754
			snap1051 := d755
			snap1052 := d756
			snap1053 := d757
			snap1054 := d758
			snap1055 := d759
			snap1056 := d760
			snap1057 := d761
			snap1058 := d762
			snap1059 := d763
			snap1060 := d764
			snap1061 := d765
			snap1062 := d766
			snap1063 := d767
			snap1064 := d768
			snap1065 := d769
			snap1066 := d770
			snap1067 := d771
			snap1068 := d772
			snap1069 := d773
			snap1070 := d774
			snap1071 := d775
			snap1072 := d776
			snap1073 := d777
			snap1074 := d778
			snap1075 := d779
			snap1076 := d780
			snap1077 := d781
			snap1078 := d782
			snap1079 := d783
			snap1080 := d784
			snap1081 := d785
			snap1082 := d786
			snap1083 := d787
			alloc1084 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps791)
			}
			ctx.RestoreAllocState(alloc1084)
			d0 = snap792
			d1 = snap793
			d2 = snap794
			d3 = snap795
			d4 = snap796
			d5 = snap797
			d6 = snap798
			d7 = snap799
			d8 = snap800
			d9 = snap801
			d10 = snap802
			d11 = snap803
			d12 = snap804
			d13 = snap805
			d14 = snap806
			d15 = snap807
			d16 = snap808
			d17 = snap809
			d18 = snap810
			d19 = snap811
			d20 = snap812
			d21 = snap813
			d22 = snap814
			d23 = snap815
			d24 = snap816
			d25 = snap817
			d26 = snap818
			d27 = snap819
			d28 = snap820
			d29 = snap821
			d30 = snap822
			d31 = snap823
			d32 = snap824
			d33 = snap825
			d34 = snap826
			d35 = snap827
			d36 = snap828
			d37 = snap829
			d38 = snap830
			d39 = snap831
			d40 = snap832
			d41 = snap833
			d42 = snap834
			d43 = snap835
			d44 = snap836
			d45 = snap837
			d46 = snap838
			d47 = snap839
			d48 = snap840
			d49 = snap841
			d50 = snap842
			d51 = snap843
			d52 = snap844
			d53 = snap845
			d54 = snap846
			d55 = snap847
			d56 = snap848
			d57 = snap849
			d58 = snap850
			d59 = snap851
			d60 = snap852
			d61 = snap853
			d62 = snap854
			d63 = snap855
			d64 = snap856
			d65 = snap857
			d66 = snap858
			d67 = snap859
			d68 = snap860
			d69 = snap861
			d70 = snap862
			d71 = snap863
			d72 = snap864
			d73 = snap865
			d74 = snap866
			d75 = snap867
			d76 = snap868
			d77 = snap869
			d78 = snap870
			d79 = snap871
			d80 = snap872
			d81 = snap873
			d82 = snap874
			d83 = snap875
			d84 = snap876
			d85 = snap877
			d86 = snap878
			d87 = snap879
			d88 = snap880
			d89 = snap881
			d90 = snap882
			d91 = snap883
			d92 = snap884
			d93 = snap885
			d94 = snap886
			d95 = snap887
			d96 = snap888
			d97 = snap889
			d98 = snap890
			d99 = snap891
			d100 = snap892
			d101 = snap893
			d102 = snap894
			d103 = snap895
			d104 = snap896
			d105 = snap897
			d106 = snap898
			d107 = snap899
			d108 = snap900
			d109 = snap901
			d110 = snap902
			d111 = snap903
			d112 = snap904
			d113 = snap905
			d114 = snap906
			d115 = snap907
			d116 = snap908
			d117 = snap909
			d118 = snap910
			d119 = snap911
			d120 = snap912
			d121 = snap913
			d122 = snap914
			d123 = snap915
			d124 = snap916
			d125 = snap917
			d126 = snap918
			d127 = snap919
			d128 = snap920
			d129 = snap921
			d130 = snap922
			d131 = snap923
			d132 = snap924
			d133 = snap925
			d134 = snap926
			d135 = snap927
			d136 = snap928
			d137 = snap929
			d138 = snap930
			d139 = snap931
			d140 = snap932
			d141 = snap933
			d142 = snap934
			d143 = snap935
			d144 = snap936
			d145 = snap937
			d146 = snap938
			d147 = snap939
			d148 = snap940
			d149 = snap941
			d150 = snap942
			d151 = snap943
			d152 = snap944
			d153 = snap945
			d154 = snap946
			d155 = snap947
			d156 = snap948
			d157 = snap949
			d158 = snap950
			d159 = snap951
			d160 = snap952
			d161 = snap953
			d162 = snap954
			d163 = snap955
			d164 = snap956
			d165 = snap957
			d166 = snap958
			d167 = snap959
			d168 = snap960
			d169 = snap961
			d170 = snap962
			d171 = snap963
			d172 = snap964
			d173 = snap965
			d174 = snap966
			d175 = snap967
			d176 = snap968
			d177 = snap969
			d178 = snap970
			d179 = snap971
			d180 = snap972
			d181 = snap973
			d182 = snap974
			d183 = snap975
			d184 = snap976
			d185 = snap977
			d186 = snap978
			d187 = snap979
			d188 = snap980
			d189 = snap981
			d190 = snap982
			d191 = snap983
			d192 = snap984
			d193 = snap985
			d194 = snap986
			d195 = snap987
			d196 = snap988
			d197 = snap989
			d198 = snap990
			d199 = snap991
			d200 = snap992
			d201 = snap993
			d202 = snap994
			d203 = snap995
			d204 = snap996
			d205 = snap997
			d206 = snap998
			d207 = snap999
			d208 = snap1000
			d209 = snap1001
			d210 = snap1002
			d211 = snap1003
			d212 = snap1004
			d213 = snap1005
			d214 = snap1006
			d215 = snap1007
			d216 = snap1008
			d217 = snap1009
			d218 = snap1010
			d219 = snap1011
			d220 = snap1012
			d221 = snap1013
			d222 = snap1014
			d223 = snap1015
			d224 = snap1016
			d225 = snap1017
			d226 = snap1018
			d227 = snap1019
			d228 = snap1020
			d229 = snap1021
			d230 = snap1022
			d231 = snap1023
			d232 = snap1024
			d233 = snap1025
			d234 = snap1026
			d235 = snap1027
			d236 = snap1028
			d237 = snap1029
			d238 = snap1030
			d239 = snap1031
			d240 = snap1032
			d487 = snap1033
			d488 = snap1034
			d489 = snap1035
			d490 = snap1036
			d741 = snap1037
			d742 = snap1038
			d743 = snap1039
			d744 = snap1040
			d745 = snap1041
			d746 = snap1042
			d747 = snap1043
			d748 = snap1044
			d749 = snap1045
			d750 = snap1046
			d751 = snap1047
			d752 = snap1048
			d753 = snap1049
			d754 = snap1050
			d755 = snap1051
			d756 = snap1052
			d757 = snap1053
			d758 = snap1054
			d759 = snap1055
			d760 = snap1056
			d761 = snap1057
			d762 = snap1058
			d763 = snap1059
			d764 = snap1060
			d765 = snap1061
			d766 = snap1062
			d767 = snap1063
			d768 = snap1064
			d769 = snap1065
			d770 = snap1066
			d771 = snap1067
			d772 = snap1068
			d773 = snap1069
			d774 = snap1070
			d775 = snap1071
			d776 = snap1072
			d777 = snap1073
			d778 = snap1074
			d779 = snap1075
			d780 = snap1076
			d781 = snap1077
			d782 = snap1078
			d783 = snap1079
			d784 = snap1080
			d785 = snap1081
			d786 = snap1082
			d787 = snap1083
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps790)
			}
			return result
			ctx.FreeDesc(&d786)
			return result
			}
			bbs[5].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[5].VisitCount >= 2 {
					ps.General = true
					return bbs[5].RenderPS(ps)
				}
			}
			bbs[5].VisitCount++
			if ps.General {
				if bbs[5].Rendered {
					ctx.EmitJmp(lbl6)
					return result
				}
				bbs[5].Rendered = true
				bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_5 = bbs[5].Address
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
				d47 = ps.OverlayValues[47]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
				d108 = ps.OverlayValues[108]
			}
			if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
				d109 = ps.OverlayValues[109]
			}
			if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
				d110 = ps.OverlayValues[110]
			}
			if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
				d111 = ps.OverlayValues[111]
			}
			if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
				d112 = ps.OverlayValues[112]
			}
			if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
				d113 = ps.OverlayValues[113]
			}
			if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
				d114 = ps.OverlayValues[114]
			}
			if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
				d115 = ps.OverlayValues[115]
			}
			if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
				d116 = ps.OverlayValues[116]
			}
			if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
				d117 = ps.OverlayValues[117]
			}
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != scm.LocNone {
				d118 = ps.OverlayValues[118]
			}
			if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
				d119 = ps.OverlayValues[119]
			}
			if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != scm.LocNone {
				d120 = ps.OverlayValues[120]
			}
			if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != scm.LocNone {
				d121 = ps.OverlayValues[121]
			}
			if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != scm.LocNone {
				d122 = ps.OverlayValues[122]
			}
			if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != scm.LocNone {
				d123 = ps.OverlayValues[123]
			}
			if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != scm.LocNone {
				d124 = ps.OverlayValues[124]
			}
			if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
				d125 = ps.OverlayValues[125]
			}
			if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
				d126 = ps.OverlayValues[126]
			}
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
				d128 = ps.OverlayValues[128]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
				d129 = ps.OverlayValues[129]
			}
			if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
				d130 = ps.OverlayValues[130]
			}
			if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
				d131 = ps.OverlayValues[131]
			}
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
				d133 = ps.OverlayValues[133]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
				d136 = ps.OverlayValues[136]
			}
			if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
				d137 = ps.OverlayValues[137]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
				d138 = ps.OverlayValues[138]
			}
			if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
				d139 = ps.OverlayValues[139]
			}
			if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
				d140 = ps.OverlayValues[140]
			}
			if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
				d141 = ps.OverlayValues[141]
			}
			if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
				d142 = ps.OverlayValues[142]
			}
			if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
				d143 = ps.OverlayValues[143]
			}
			if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
				d144 = ps.OverlayValues[144]
			}
			if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
				d145 = ps.OverlayValues[145]
			}
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
				d146 = ps.OverlayValues[146]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
				d155 = ps.OverlayValues[155]
			}
			if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
				d156 = ps.OverlayValues[156]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
				d159 = ps.OverlayValues[159]
			}
			if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
				d160 = ps.OverlayValues[160]
			}
			if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
				d161 = ps.OverlayValues[161]
			}
			if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
				d162 = ps.OverlayValues[162]
			}
			if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
				d163 = ps.OverlayValues[163]
			}
			if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
				d164 = ps.OverlayValues[164]
			}
			if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
				d165 = ps.OverlayValues[165]
			}
			if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
				d166 = ps.OverlayValues[166]
			}
			if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
				d167 = ps.OverlayValues[167]
			}
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
				d169 = ps.OverlayValues[169]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != scm.LocNone {
				d172 = ps.OverlayValues[172]
			}
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
				d176 = ps.OverlayValues[176]
			}
			if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
				d177 = ps.OverlayValues[177]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
				d179 = ps.OverlayValues[179]
			}
			if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
				d180 = ps.OverlayValues[180]
			}
			if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
				d181 = ps.OverlayValues[181]
			}
			if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
				d182 = ps.OverlayValues[182]
			}
			if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
				d183 = ps.OverlayValues[183]
			}
			if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
				d184 = ps.OverlayValues[184]
			}
			if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
				d185 = ps.OverlayValues[185]
			}
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
				d191 = ps.OverlayValues[191]
			}
			if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
				d192 = ps.OverlayValues[192]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
				d194 = ps.OverlayValues[194]
			}
			if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
				d195 = ps.OverlayValues[195]
			}
			if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
				d196 = ps.OverlayValues[196]
			}
			if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
				d197 = ps.OverlayValues[197]
			}
			if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
				d198 = ps.OverlayValues[198]
			}
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
				d205 = ps.OverlayValues[205]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != scm.LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != scm.LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != scm.LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != scm.LocNone {
				d211 = ps.OverlayValues[211]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != scm.LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != scm.LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != scm.LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != scm.LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != scm.LocNone {
				d216 = ps.OverlayValues[216]
			}
			if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != scm.LocNone {
				d217 = ps.OverlayValues[217]
			}
			if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != scm.LocNone {
				d218 = ps.OverlayValues[218]
			}
			if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != scm.LocNone {
				d219 = ps.OverlayValues[219]
			}
			if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
				d220 = ps.OverlayValues[220]
			}
			if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != scm.LocNone {
				d221 = ps.OverlayValues[221]
			}
			if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != scm.LocNone {
				d222 = ps.OverlayValues[222]
			}
			if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
				d223 = ps.OverlayValues[223]
			}
			if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != scm.LocNone {
				d224 = ps.OverlayValues[224]
			}
			if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
				d225 = ps.OverlayValues[225]
			}
			if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
				d226 = ps.OverlayValues[226]
			}
			if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
				d227 = ps.OverlayValues[227]
			}
			if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
				d228 = ps.OverlayValues[228]
			}
			if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
				d229 = ps.OverlayValues[229]
			}
			if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
				d230 = ps.OverlayValues[230]
			}
			if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
				d231 = ps.OverlayValues[231]
			}
			if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
				d232 = ps.OverlayValues[232]
			}
			if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
				d233 = ps.OverlayValues[233]
			}
			if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
				d234 = ps.OverlayValues[234]
			}
			if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
				d235 = ps.OverlayValues[235]
			}
			if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
				d236 = ps.OverlayValues[236]
			}
			if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
				d237 = ps.OverlayValues[237]
			}
			if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
				d238 = ps.OverlayValues[238]
			}
			if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
				d239 = ps.OverlayValues[239]
			}
			if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
				d240 = ps.OverlayValues[240]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			if len(ps.OverlayValues) > 741 && ps.OverlayValues[741].Loc != scm.LocNone {
				d741 = ps.OverlayValues[741]
			}
			if len(ps.OverlayValues) > 742 && ps.OverlayValues[742].Loc != scm.LocNone {
				d742 = ps.OverlayValues[742]
			}
			if len(ps.OverlayValues) > 743 && ps.OverlayValues[743].Loc != scm.LocNone {
				d743 = ps.OverlayValues[743]
			}
			if len(ps.OverlayValues) > 744 && ps.OverlayValues[744].Loc != scm.LocNone {
				d744 = ps.OverlayValues[744]
			}
			if len(ps.OverlayValues) > 745 && ps.OverlayValues[745].Loc != scm.LocNone {
				d745 = ps.OverlayValues[745]
			}
			if len(ps.OverlayValues) > 746 && ps.OverlayValues[746].Loc != scm.LocNone {
				d746 = ps.OverlayValues[746]
			}
			if len(ps.OverlayValues) > 747 && ps.OverlayValues[747].Loc != scm.LocNone {
				d747 = ps.OverlayValues[747]
			}
			if len(ps.OverlayValues) > 748 && ps.OverlayValues[748].Loc != scm.LocNone {
				d748 = ps.OverlayValues[748]
			}
			if len(ps.OverlayValues) > 749 && ps.OverlayValues[749].Loc != scm.LocNone {
				d749 = ps.OverlayValues[749]
			}
			if len(ps.OverlayValues) > 750 && ps.OverlayValues[750].Loc != scm.LocNone {
				d750 = ps.OverlayValues[750]
			}
			if len(ps.OverlayValues) > 751 && ps.OverlayValues[751].Loc != scm.LocNone {
				d751 = ps.OverlayValues[751]
			}
			if len(ps.OverlayValues) > 752 && ps.OverlayValues[752].Loc != scm.LocNone {
				d752 = ps.OverlayValues[752]
			}
			if len(ps.OverlayValues) > 753 && ps.OverlayValues[753].Loc != scm.LocNone {
				d753 = ps.OverlayValues[753]
			}
			if len(ps.OverlayValues) > 754 && ps.OverlayValues[754].Loc != scm.LocNone {
				d754 = ps.OverlayValues[754]
			}
			if len(ps.OverlayValues) > 755 && ps.OverlayValues[755].Loc != scm.LocNone {
				d755 = ps.OverlayValues[755]
			}
			if len(ps.OverlayValues) > 756 && ps.OverlayValues[756].Loc != scm.LocNone {
				d756 = ps.OverlayValues[756]
			}
			if len(ps.OverlayValues) > 757 && ps.OverlayValues[757].Loc != scm.LocNone {
				d757 = ps.OverlayValues[757]
			}
			if len(ps.OverlayValues) > 758 && ps.OverlayValues[758].Loc != scm.LocNone {
				d758 = ps.OverlayValues[758]
			}
			if len(ps.OverlayValues) > 759 && ps.OverlayValues[759].Loc != scm.LocNone {
				d759 = ps.OverlayValues[759]
			}
			if len(ps.OverlayValues) > 760 && ps.OverlayValues[760].Loc != scm.LocNone {
				d760 = ps.OverlayValues[760]
			}
			if len(ps.OverlayValues) > 761 && ps.OverlayValues[761].Loc != scm.LocNone {
				d761 = ps.OverlayValues[761]
			}
			if len(ps.OverlayValues) > 762 && ps.OverlayValues[762].Loc != scm.LocNone {
				d762 = ps.OverlayValues[762]
			}
			if len(ps.OverlayValues) > 763 && ps.OverlayValues[763].Loc != scm.LocNone {
				d763 = ps.OverlayValues[763]
			}
			if len(ps.OverlayValues) > 764 && ps.OverlayValues[764].Loc != scm.LocNone {
				d764 = ps.OverlayValues[764]
			}
			if len(ps.OverlayValues) > 765 && ps.OverlayValues[765].Loc != scm.LocNone {
				d765 = ps.OverlayValues[765]
			}
			if len(ps.OverlayValues) > 766 && ps.OverlayValues[766].Loc != scm.LocNone {
				d766 = ps.OverlayValues[766]
			}
			if len(ps.OverlayValues) > 767 && ps.OverlayValues[767].Loc != scm.LocNone {
				d767 = ps.OverlayValues[767]
			}
			if len(ps.OverlayValues) > 768 && ps.OverlayValues[768].Loc != scm.LocNone {
				d768 = ps.OverlayValues[768]
			}
			if len(ps.OverlayValues) > 769 && ps.OverlayValues[769].Loc != scm.LocNone {
				d769 = ps.OverlayValues[769]
			}
			if len(ps.OverlayValues) > 770 && ps.OverlayValues[770].Loc != scm.LocNone {
				d770 = ps.OverlayValues[770]
			}
			if len(ps.OverlayValues) > 771 && ps.OverlayValues[771].Loc != scm.LocNone {
				d771 = ps.OverlayValues[771]
			}
			if len(ps.OverlayValues) > 772 && ps.OverlayValues[772].Loc != scm.LocNone {
				d772 = ps.OverlayValues[772]
			}
			if len(ps.OverlayValues) > 773 && ps.OverlayValues[773].Loc != scm.LocNone {
				d773 = ps.OverlayValues[773]
			}
			if len(ps.OverlayValues) > 774 && ps.OverlayValues[774].Loc != scm.LocNone {
				d774 = ps.OverlayValues[774]
			}
			if len(ps.OverlayValues) > 775 && ps.OverlayValues[775].Loc != scm.LocNone {
				d775 = ps.OverlayValues[775]
			}
			if len(ps.OverlayValues) > 776 && ps.OverlayValues[776].Loc != scm.LocNone {
				d776 = ps.OverlayValues[776]
			}
			if len(ps.OverlayValues) > 777 && ps.OverlayValues[777].Loc != scm.LocNone {
				d777 = ps.OverlayValues[777]
			}
			if len(ps.OverlayValues) > 778 && ps.OverlayValues[778].Loc != scm.LocNone {
				d778 = ps.OverlayValues[778]
			}
			if len(ps.OverlayValues) > 779 && ps.OverlayValues[779].Loc != scm.LocNone {
				d779 = ps.OverlayValues[779]
			}
			if len(ps.OverlayValues) > 780 && ps.OverlayValues[780].Loc != scm.LocNone {
				d780 = ps.OverlayValues[780]
			}
			if len(ps.OverlayValues) > 781 && ps.OverlayValues[781].Loc != scm.LocNone {
				d781 = ps.OverlayValues[781]
			}
			if len(ps.OverlayValues) > 782 && ps.OverlayValues[782].Loc != scm.LocNone {
				d782 = ps.OverlayValues[782]
			}
			if len(ps.OverlayValues) > 783 && ps.OverlayValues[783].Loc != scm.LocNone {
				d783 = ps.OverlayValues[783]
			}
			if len(ps.OverlayValues) > 784 && ps.OverlayValues[784].Loc != scm.LocNone {
				d784 = ps.OverlayValues[784]
			}
			if len(ps.OverlayValues) > 785 && ps.OverlayValues[785].Loc != scm.LocNone {
				d785 = ps.OverlayValues[785]
			}
			if len(ps.OverlayValues) > 786 && ps.OverlayValues[786].Loc != scm.LocNone {
				d786 = ps.OverlayValues[786]
			}
			if len(ps.OverlayValues) > 787 && ps.OverlayValues[787].Loc != scm.LocNone {
				d787 = ps.OverlayValues[787]
			}
			ctx.ReclaimUntrackedRegs()
			d1085 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewString("prefix index out of range")}
			ctx.EnsureDesc(&d1085)
			ctx.EnsureDesc(&d1085)
			if d1085.Loc == scm.LocImm {
				tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1085.Imm.GetTag() == scm.TagBool {
					ctx.EmitMakeBool(tmpPair, d1085)
				} else if d1085.Imm.GetTag() == scm.TagInt {
					ctx.EmitMakeInt(tmpPair, d1085)
				} else if d1085.Imm.GetTag() == scm.TagFloat {
					ctx.EmitMakeFloat(tmpPair, d1085)
				} else if d1085.Imm.GetTag() == scm.TagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1085.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1085 = tmpPair
			} else if d1085.Loc == scm.LocReg {
				tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: d1085.Type, Reg: ctx.AllocRegExcept(d1085.Reg), Reg2: ctx.AllocRegExcept(d1085.Reg)}
				switch d1085.Type {
				case scm.TagBool:
					ctx.EmitMakeBool(tmpPair, d1085)
				case scm.TagInt:
					ctx.EmitMakeInt(tmpPair, d1085)
				case scm.TagFloat:
					ctx.EmitMakeFloat(tmpPair, d1085)
				default:
					panic("jit: panic arg scalar type unknown for scm.Scmer pair")
				}
				ctx.FreeDesc(&d1085)
				d1085 = tmpPair
			}
			if d1085.Loc != scm.LocRegPair && d1085.Loc != scm.LocStackPair {
				panic("jit: panic arg expects scm.Scmer pair")
			}
			ctx.EmitGoCallVoid(scm.GoFuncAddr(scm.JITPanic), []scm.JITValueDesc{d1085})
			ctx.FreeDesc(&d1085)
			return result
			}
			bbs[6].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[6].VisitCount >= 2 {
					ps.General = true
					return bbs[6].RenderPS(ps)
				}
			}
			bbs[6].VisitCount++
			if ps.General {
				if bbs[6].Rendered {
					ctx.EmitJmp(lbl7)
					return result
				}
				bbs[6].Rendered = true
				bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_6 = bbs[6].Address
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
				d47 = ps.OverlayValues[47]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
				d108 = ps.OverlayValues[108]
			}
			if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
				d109 = ps.OverlayValues[109]
			}
			if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
				d110 = ps.OverlayValues[110]
			}
			if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
				d111 = ps.OverlayValues[111]
			}
			if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
				d112 = ps.OverlayValues[112]
			}
			if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
				d113 = ps.OverlayValues[113]
			}
			if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
				d114 = ps.OverlayValues[114]
			}
			if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
				d115 = ps.OverlayValues[115]
			}
			if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
				d116 = ps.OverlayValues[116]
			}
			if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
				d117 = ps.OverlayValues[117]
			}
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != scm.LocNone {
				d118 = ps.OverlayValues[118]
			}
			if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
				d119 = ps.OverlayValues[119]
			}
			if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != scm.LocNone {
				d120 = ps.OverlayValues[120]
			}
			if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != scm.LocNone {
				d121 = ps.OverlayValues[121]
			}
			if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != scm.LocNone {
				d122 = ps.OverlayValues[122]
			}
			if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != scm.LocNone {
				d123 = ps.OverlayValues[123]
			}
			if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != scm.LocNone {
				d124 = ps.OverlayValues[124]
			}
			if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
				d125 = ps.OverlayValues[125]
			}
			if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
				d126 = ps.OverlayValues[126]
			}
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
				d128 = ps.OverlayValues[128]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
				d129 = ps.OverlayValues[129]
			}
			if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
				d130 = ps.OverlayValues[130]
			}
			if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
				d131 = ps.OverlayValues[131]
			}
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
				d133 = ps.OverlayValues[133]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
				d136 = ps.OverlayValues[136]
			}
			if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
				d137 = ps.OverlayValues[137]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
				d138 = ps.OverlayValues[138]
			}
			if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
				d139 = ps.OverlayValues[139]
			}
			if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
				d140 = ps.OverlayValues[140]
			}
			if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
				d141 = ps.OverlayValues[141]
			}
			if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
				d142 = ps.OverlayValues[142]
			}
			if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
				d143 = ps.OverlayValues[143]
			}
			if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
				d144 = ps.OverlayValues[144]
			}
			if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
				d145 = ps.OverlayValues[145]
			}
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
				d146 = ps.OverlayValues[146]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
				d155 = ps.OverlayValues[155]
			}
			if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
				d156 = ps.OverlayValues[156]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
				d159 = ps.OverlayValues[159]
			}
			if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
				d160 = ps.OverlayValues[160]
			}
			if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
				d161 = ps.OverlayValues[161]
			}
			if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
				d162 = ps.OverlayValues[162]
			}
			if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
				d163 = ps.OverlayValues[163]
			}
			if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
				d164 = ps.OverlayValues[164]
			}
			if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
				d165 = ps.OverlayValues[165]
			}
			if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
				d166 = ps.OverlayValues[166]
			}
			if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
				d167 = ps.OverlayValues[167]
			}
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
				d169 = ps.OverlayValues[169]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != scm.LocNone {
				d172 = ps.OverlayValues[172]
			}
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
				d176 = ps.OverlayValues[176]
			}
			if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
				d177 = ps.OverlayValues[177]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
				d179 = ps.OverlayValues[179]
			}
			if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
				d180 = ps.OverlayValues[180]
			}
			if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
				d181 = ps.OverlayValues[181]
			}
			if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
				d182 = ps.OverlayValues[182]
			}
			if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
				d183 = ps.OverlayValues[183]
			}
			if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
				d184 = ps.OverlayValues[184]
			}
			if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
				d185 = ps.OverlayValues[185]
			}
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
				d191 = ps.OverlayValues[191]
			}
			if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
				d192 = ps.OverlayValues[192]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
				d194 = ps.OverlayValues[194]
			}
			if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
				d195 = ps.OverlayValues[195]
			}
			if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
				d196 = ps.OverlayValues[196]
			}
			if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
				d197 = ps.OverlayValues[197]
			}
			if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
				d198 = ps.OverlayValues[198]
			}
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
				d205 = ps.OverlayValues[205]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != scm.LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != scm.LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != scm.LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != scm.LocNone {
				d211 = ps.OverlayValues[211]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != scm.LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != scm.LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != scm.LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != scm.LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != scm.LocNone {
				d216 = ps.OverlayValues[216]
			}
			if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != scm.LocNone {
				d217 = ps.OverlayValues[217]
			}
			if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != scm.LocNone {
				d218 = ps.OverlayValues[218]
			}
			if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != scm.LocNone {
				d219 = ps.OverlayValues[219]
			}
			if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
				d220 = ps.OverlayValues[220]
			}
			if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != scm.LocNone {
				d221 = ps.OverlayValues[221]
			}
			if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != scm.LocNone {
				d222 = ps.OverlayValues[222]
			}
			if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
				d223 = ps.OverlayValues[223]
			}
			if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != scm.LocNone {
				d224 = ps.OverlayValues[224]
			}
			if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
				d225 = ps.OverlayValues[225]
			}
			if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
				d226 = ps.OverlayValues[226]
			}
			if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
				d227 = ps.OverlayValues[227]
			}
			if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
				d228 = ps.OverlayValues[228]
			}
			if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
				d229 = ps.OverlayValues[229]
			}
			if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
				d230 = ps.OverlayValues[230]
			}
			if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
				d231 = ps.OverlayValues[231]
			}
			if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
				d232 = ps.OverlayValues[232]
			}
			if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
				d233 = ps.OverlayValues[233]
			}
			if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
				d234 = ps.OverlayValues[234]
			}
			if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
				d235 = ps.OverlayValues[235]
			}
			if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
				d236 = ps.OverlayValues[236]
			}
			if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
				d237 = ps.OverlayValues[237]
			}
			if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
				d238 = ps.OverlayValues[238]
			}
			if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
				d239 = ps.OverlayValues[239]
			}
			if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
				d240 = ps.OverlayValues[240]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			if len(ps.OverlayValues) > 741 && ps.OverlayValues[741].Loc != scm.LocNone {
				d741 = ps.OverlayValues[741]
			}
			if len(ps.OverlayValues) > 742 && ps.OverlayValues[742].Loc != scm.LocNone {
				d742 = ps.OverlayValues[742]
			}
			if len(ps.OverlayValues) > 743 && ps.OverlayValues[743].Loc != scm.LocNone {
				d743 = ps.OverlayValues[743]
			}
			if len(ps.OverlayValues) > 744 && ps.OverlayValues[744].Loc != scm.LocNone {
				d744 = ps.OverlayValues[744]
			}
			if len(ps.OverlayValues) > 745 && ps.OverlayValues[745].Loc != scm.LocNone {
				d745 = ps.OverlayValues[745]
			}
			if len(ps.OverlayValues) > 746 && ps.OverlayValues[746].Loc != scm.LocNone {
				d746 = ps.OverlayValues[746]
			}
			if len(ps.OverlayValues) > 747 && ps.OverlayValues[747].Loc != scm.LocNone {
				d747 = ps.OverlayValues[747]
			}
			if len(ps.OverlayValues) > 748 && ps.OverlayValues[748].Loc != scm.LocNone {
				d748 = ps.OverlayValues[748]
			}
			if len(ps.OverlayValues) > 749 && ps.OverlayValues[749].Loc != scm.LocNone {
				d749 = ps.OverlayValues[749]
			}
			if len(ps.OverlayValues) > 750 && ps.OverlayValues[750].Loc != scm.LocNone {
				d750 = ps.OverlayValues[750]
			}
			if len(ps.OverlayValues) > 751 && ps.OverlayValues[751].Loc != scm.LocNone {
				d751 = ps.OverlayValues[751]
			}
			if len(ps.OverlayValues) > 752 && ps.OverlayValues[752].Loc != scm.LocNone {
				d752 = ps.OverlayValues[752]
			}
			if len(ps.OverlayValues) > 753 && ps.OverlayValues[753].Loc != scm.LocNone {
				d753 = ps.OverlayValues[753]
			}
			if len(ps.OverlayValues) > 754 && ps.OverlayValues[754].Loc != scm.LocNone {
				d754 = ps.OverlayValues[754]
			}
			if len(ps.OverlayValues) > 755 && ps.OverlayValues[755].Loc != scm.LocNone {
				d755 = ps.OverlayValues[755]
			}
			if len(ps.OverlayValues) > 756 && ps.OverlayValues[756].Loc != scm.LocNone {
				d756 = ps.OverlayValues[756]
			}
			if len(ps.OverlayValues) > 757 && ps.OverlayValues[757].Loc != scm.LocNone {
				d757 = ps.OverlayValues[757]
			}
			if len(ps.OverlayValues) > 758 && ps.OverlayValues[758].Loc != scm.LocNone {
				d758 = ps.OverlayValues[758]
			}
			if len(ps.OverlayValues) > 759 && ps.OverlayValues[759].Loc != scm.LocNone {
				d759 = ps.OverlayValues[759]
			}
			if len(ps.OverlayValues) > 760 && ps.OverlayValues[760].Loc != scm.LocNone {
				d760 = ps.OverlayValues[760]
			}
			if len(ps.OverlayValues) > 761 && ps.OverlayValues[761].Loc != scm.LocNone {
				d761 = ps.OverlayValues[761]
			}
			if len(ps.OverlayValues) > 762 && ps.OverlayValues[762].Loc != scm.LocNone {
				d762 = ps.OverlayValues[762]
			}
			if len(ps.OverlayValues) > 763 && ps.OverlayValues[763].Loc != scm.LocNone {
				d763 = ps.OverlayValues[763]
			}
			if len(ps.OverlayValues) > 764 && ps.OverlayValues[764].Loc != scm.LocNone {
				d764 = ps.OverlayValues[764]
			}
			if len(ps.OverlayValues) > 765 && ps.OverlayValues[765].Loc != scm.LocNone {
				d765 = ps.OverlayValues[765]
			}
			if len(ps.OverlayValues) > 766 && ps.OverlayValues[766].Loc != scm.LocNone {
				d766 = ps.OverlayValues[766]
			}
			if len(ps.OverlayValues) > 767 && ps.OverlayValues[767].Loc != scm.LocNone {
				d767 = ps.OverlayValues[767]
			}
			if len(ps.OverlayValues) > 768 && ps.OverlayValues[768].Loc != scm.LocNone {
				d768 = ps.OverlayValues[768]
			}
			if len(ps.OverlayValues) > 769 && ps.OverlayValues[769].Loc != scm.LocNone {
				d769 = ps.OverlayValues[769]
			}
			if len(ps.OverlayValues) > 770 && ps.OverlayValues[770].Loc != scm.LocNone {
				d770 = ps.OverlayValues[770]
			}
			if len(ps.OverlayValues) > 771 && ps.OverlayValues[771].Loc != scm.LocNone {
				d771 = ps.OverlayValues[771]
			}
			if len(ps.OverlayValues) > 772 && ps.OverlayValues[772].Loc != scm.LocNone {
				d772 = ps.OverlayValues[772]
			}
			if len(ps.OverlayValues) > 773 && ps.OverlayValues[773].Loc != scm.LocNone {
				d773 = ps.OverlayValues[773]
			}
			if len(ps.OverlayValues) > 774 && ps.OverlayValues[774].Loc != scm.LocNone {
				d774 = ps.OverlayValues[774]
			}
			if len(ps.OverlayValues) > 775 && ps.OverlayValues[775].Loc != scm.LocNone {
				d775 = ps.OverlayValues[775]
			}
			if len(ps.OverlayValues) > 776 && ps.OverlayValues[776].Loc != scm.LocNone {
				d776 = ps.OverlayValues[776]
			}
			if len(ps.OverlayValues) > 777 && ps.OverlayValues[777].Loc != scm.LocNone {
				d777 = ps.OverlayValues[777]
			}
			if len(ps.OverlayValues) > 778 && ps.OverlayValues[778].Loc != scm.LocNone {
				d778 = ps.OverlayValues[778]
			}
			if len(ps.OverlayValues) > 779 && ps.OverlayValues[779].Loc != scm.LocNone {
				d779 = ps.OverlayValues[779]
			}
			if len(ps.OverlayValues) > 780 && ps.OverlayValues[780].Loc != scm.LocNone {
				d780 = ps.OverlayValues[780]
			}
			if len(ps.OverlayValues) > 781 && ps.OverlayValues[781].Loc != scm.LocNone {
				d781 = ps.OverlayValues[781]
			}
			if len(ps.OverlayValues) > 782 && ps.OverlayValues[782].Loc != scm.LocNone {
				d782 = ps.OverlayValues[782]
			}
			if len(ps.OverlayValues) > 783 && ps.OverlayValues[783].Loc != scm.LocNone {
				d783 = ps.OverlayValues[783]
			}
			if len(ps.OverlayValues) > 784 && ps.OverlayValues[784].Loc != scm.LocNone {
				d784 = ps.OverlayValues[784]
			}
			if len(ps.OverlayValues) > 785 && ps.OverlayValues[785].Loc != scm.LocNone {
				d785 = ps.OverlayValues[785]
			}
			if len(ps.OverlayValues) > 786 && ps.OverlayValues[786].Loc != scm.LocNone {
				d786 = ps.OverlayValues[786]
			}
			if len(ps.OverlayValues) > 787 && ps.OverlayValues[787].Loc != scm.LocNone {
				d787 = ps.OverlayValues[787]
			}
			if len(ps.OverlayValues) > 1085 && ps.OverlayValues[1085].Loc != scm.LocNone {
				d1085 = ps.OverlayValues[1085]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d782)
			r295 := ctx.AllocReg()
			ctx.EnsureDesc(&d782)
			ctx.EnsureDesc(&d783)
			if d782.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r295, uint64(d782.Imm.Int()) * 16)
			} else {
				ctx.EmitMovRegReg(r295, d782.Reg)
				ctx.EmitShlRegImm8(r295, 4)
			}
			if d783.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d783.Imm.Int()))
				ctx.EmitAddInt64(r295, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r295, d783.Reg)
			}
			r296 := ctx.AllocRegExcept(r295)
			r297 := ctx.AllocRegExcept(r295, r296)
			ctx.EmitMovRegMem(r296, r295, 0)
			ctx.EmitMovRegMem(r297, r295, 8)
			ctx.FreeReg(r295)
			d1086 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r296, Reg2: r297}
			ctx.BindReg(r296, &d1086)
			ctx.BindReg(r297, &d1086)
			d1088 = d237
			ctx.EnsureDesc(&d1088)
			if d1088.Loc == scm.LocImm {
				tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d1088.Imm.GetTag()
				switch tag {
				case scm.TagBool:
					ctx.EmitMakeBool(tmpPair, d1088)
				case scm.TagInt:
					ctx.EmitMakeInt(tmpPair, d1088)
				case scm.TagFloat:
					ctx.EmitMakeFloat(tmpPair, d1088)
				case scm.TagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d1088.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1088 = tmpPair
			} else if d1088.Loc == scm.LocReg {
				tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocRegExcept(d1088.Reg), Reg2: ctx.AllocRegExcept(d1088.Reg)}
				switch d1088.Type {
				case scm.TagBool:
					ctx.EmitMakeBool(tmpPair, d1088)
				case scm.TagInt:
					ctx.EmitMakeInt(tmpPair, d1088)
				case scm.TagFloat:
					ctx.EmitMakeFloat(tmpPair, d1088)
				default:
					panic("jit: scm.Scmer.String requires scm.Scmer pair receiver")
				}
				ctx.FreeDesc(&d1088)
				d1088 = tmpPair
			} else if d1088.Loc == scm.LocMem {
				tmpScalar := scm.JITValueDesc{Loc: scm.LocReg, Type: d1088.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d1088.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case scm.TagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case scm.TagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case scm.TagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: scm.Scmer.String requires scm.Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d1088 = tmpPair
			}
			if d1088.Loc != scm.LocRegPair && d1088.Loc != scm.LocStackPair {
				panic("jit: scm.Scmer.String receiver not materialized as pair")
			}
			d1087 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d1088}, 2)
			ctx.FreeDesc(&d237)
			ctx.EnsureDesc(&d1086)
			ctx.EnsureDesc(&d1087)
			d1089 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d1086, d1087}, 2)
			ctx.FreeDesc(&d1086)
			d1090 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d1090)
			ctx.BindReg(r1, &d1090)
			d1091 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d1089}, 2)
			ctx.EmitMovPairToResult(&d1091, &d1090)
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[7].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[7].VisitCount >= 2 {
					ps.General = true
					return bbs[7].RenderPS(ps)
				}
			}
			bbs[7].VisitCount++
			if ps.General {
				if bbs[7].Rendered {
					ctx.EmitJmp(lbl8)
					return result
				}
				bbs[7].Rendered = true
				bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_7 = bbs[7].Address
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
				d47 = ps.OverlayValues[47]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
				d108 = ps.OverlayValues[108]
			}
			if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
				d109 = ps.OverlayValues[109]
			}
			if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
				d110 = ps.OverlayValues[110]
			}
			if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
				d111 = ps.OverlayValues[111]
			}
			if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
				d112 = ps.OverlayValues[112]
			}
			if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
				d113 = ps.OverlayValues[113]
			}
			if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
				d114 = ps.OverlayValues[114]
			}
			if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
				d115 = ps.OverlayValues[115]
			}
			if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
				d116 = ps.OverlayValues[116]
			}
			if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
				d117 = ps.OverlayValues[117]
			}
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != scm.LocNone {
				d118 = ps.OverlayValues[118]
			}
			if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
				d119 = ps.OverlayValues[119]
			}
			if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != scm.LocNone {
				d120 = ps.OverlayValues[120]
			}
			if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != scm.LocNone {
				d121 = ps.OverlayValues[121]
			}
			if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != scm.LocNone {
				d122 = ps.OverlayValues[122]
			}
			if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != scm.LocNone {
				d123 = ps.OverlayValues[123]
			}
			if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != scm.LocNone {
				d124 = ps.OverlayValues[124]
			}
			if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
				d125 = ps.OverlayValues[125]
			}
			if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
				d126 = ps.OverlayValues[126]
			}
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
				d128 = ps.OverlayValues[128]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
				d129 = ps.OverlayValues[129]
			}
			if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
				d130 = ps.OverlayValues[130]
			}
			if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
				d131 = ps.OverlayValues[131]
			}
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
				d133 = ps.OverlayValues[133]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
				d136 = ps.OverlayValues[136]
			}
			if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
				d137 = ps.OverlayValues[137]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
				d138 = ps.OverlayValues[138]
			}
			if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
				d139 = ps.OverlayValues[139]
			}
			if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
				d140 = ps.OverlayValues[140]
			}
			if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
				d141 = ps.OverlayValues[141]
			}
			if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
				d142 = ps.OverlayValues[142]
			}
			if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
				d143 = ps.OverlayValues[143]
			}
			if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
				d144 = ps.OverlayValues[144]
			}
			if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
				d145 = ps.OverlayValues[145]
			}
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
				d146 = ps.OverlayValues[146]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
				d155 = ps.OverlayValues[155]
			}
			if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
				d156 = ps.OverlayValues[156]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
				d159 = ps.OverlayValues[159]
			}
			if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
				d160 = ps.OverlayValues[160]
			}
			if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
				d161 = ps.OverlayValues[161]
			}
			if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
				d162 = ps.OverlayValues[162]
			}
			if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
				d163 = ps.OverlayValues[163]
			}
			if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
				d164 = ps.OverlayValues[164]
			}
			if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
				d165 = ps.OverlayValues[165]
			}
			if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
				d166 = ps.OverlayValues[166]
			}
			if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
				d167 = ps.OverlayValues[167]
			}
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
				d169 = ps.OverlayValues[169]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != scm.LocNone {
				d172 = ps.OverlayValues[172]
			}
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
				d176 = ps.OverlayValues[176]
			}
			if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
				d177 = ps.OverlayValues[177]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
				d179 = ps.OverlayValues[179]
			}
			if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
				d180 = ps.OverlayValues[180]
			}
			if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
				d181 = ps.OverlayValues[181]
			}
			if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
				d182 = ps.OverlayValues[182]
			}
			if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
				d183 = ps.OverlayValues[183]
			}
			if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
				d184 = ps.OverlayValues[184]
			}
			if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
				d185 = ps.OverlayValues[185]
			}
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
				d191 = ps.OverlayValues[191]
			}
			if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
				d192 = ps.OverlayValues[192]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
				d194 = ps.OverlayValues[194]
			}
			if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
				d195 = ps.OverlayValues[195]
			}
			if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
				d196 = ps.OverlayValues[196]
			}
			if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
				d197 = ps.OverlayValues[197]
			}
			if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
				d198 = ps.OverlayValues[198]
			}
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
				d205 = ps.OverlayValues[205]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != scm.LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != scm.LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != scm.LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != scm.LocNone {
				d211 = ps.OverlayValues[211]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != scm.LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != scm.LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != scm.LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != scm.LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != scm.LocNone {
				d216 = ps.OverlayValues[216]
			}
			if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != scm.LocNone {
				d217 = ps.OverlayValues[217]
			}
			if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != scm.LocNone {
				d218 = ps.OverlayValues[218]
			}
			if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != scm.LocNone {
				d219 = ps.OverlayValues[219]
			}
			if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
				d220 = ps.OverlayValues[220]
			}
			if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != scm.LocNone {
				d221 = ps.OverlayValues[221]
			}
			if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != scm.LocNone {
				d222 = ps.OverlayValues[222]
			}
			if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
				d223 = ps.OverlayValues[223]
			}
			if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != scm.LocNone {
				d224 = ps.OverlayValues[224]
			}
			if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
				d225 = ps.OverlayValues[225]
			}
			if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
				d226 = ps.OverlayValues[226]
			}
			if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
				d227 = ps.OverlayValues[227]
			}
			if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
				d228 = ps.OverlayValues[228]
			}
			if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
				d229 = ps.OverlayValues[229]
			}
			if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
				d230 = ps.OverlayValues[230]
			}
			if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
				d231 = ps.OverlayValues[231]
			}
			if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
				d232 = ps.OverlayValues[232]
			}
			if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
				d233 = ps.OverlayValues[233]
			}
			if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
				d234 = ps.OverlayValues[234]
			}
			if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
				d235 = ps.OverlayValues[235]
			}
			if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
				d236 = ps.OverlayValues[236]
			}
			if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
				d237 = ps.OverlayValues[237]
			}
			if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
				d238 = ps.OverlayValues[238]
			}
			if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
				d239 = ps.OverlayValues[239]
			}
			if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
				d240 = ps.OverlayValues[240]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			if len(ps.OverlayValues) > 741 && ps.OverlayValues[741].Loc != scm.LocNone {
				d741 = ps.OverlayValues[741]
			}
			if len(ps.OverlayValues) > 742 && ps.OverlayValues[742].Loc != scm.LocNone {
				d742 = ps.OverlayValues[742]
			}
			if len(ps.OverlayValues) > 743 && ps.OverlayValues[743].Loc != scm.LocNone {
				d743 = ps.OverlayValues[743]
			}
			if len(ps.OverlayValues) > 744 && ps.OverlayValues[744].Loc != scm.LocNone {
				d744 = ps.OverlayValues[744]
			}
			if len(ps.OverlayValues) > 745 && ps.OverlayValues[745].Loc != scm.LocNone {
				d745 = ps.OverlayValues[745]
			}
			if len(ps.OverlayValues) > 746 && ps.OverlayValues[746].Loc != scm.LocNone {
				d746 = ps.OverlayValues[746]
			}
			if len(ps.OverlayValues) > 747 && ps.OverlayValues[747].Loc != scm.LocNone {
				d747 = ps.OverlayValues[747]
			}
			if len(ps.OverlayValues) > 748 && ps.OverlayValues[748].Loc != scm.LocNone {
				d748 = ps.OverlayValues[748]
			}
			if len(ps.OverlayValues) > 749 && ps.OverlayValues[749].Loc != scm.LocNone {
				d749 = ps.OverlayValues[749]
			}
			if len(ps.OverlayValues) > 750 && ps.OverlayValues[750].Loc != scm.LocNone {
				d750 = ps.OverlayValues[750]
			}
			if len(ps.OverlayValues) > 751 && ps.OverlayValues[751].Loc != scm.LocNone {
				d751 = ps.OverlayValues[751]
			}
			if len(ps.OverlayValues) > 752 && ps.OverlayValues[752].Loc != scm.LocNone {
				d752 = ps.OverlayValues[752]
			}
			if len(ps.OverlayValues) > 753 && ps.OverlayValues[753].Loc != scm.LocNone {
				d753 = ps.OverlayValues[753]
			}
			if len(ps.OverlayValues) > 754 && ps.OverlayValues[754].Loc != scm.LocNone {
				d754 = ps.OverlayValues[754]
			}
			if len(ps.OverlayValues) > 755 && ps.OverlayValues[755].Loc != scm.LocNone {
				d755 = ps.OverlayValues[755]
			}
			if len(ps.OverlayValues) > 756 && ps.OverlayValues[756].Loc != scm.LocNone {
				d756 = ps.OverlayValues[756]
			}
			if len(ps.OverlayValues) > 757 && ps.OverlayValues[757].Loc != scm.LocNone {
				d757 = ps.OverlayValues[757]
			}
			if len(ps.OverlayValues) > 758 && ps.OverlayValues[758].Loc != scm.LocNone {
				d758 = ps.OverlayValues[758]
			}
			if len(ps.OverlayValues) > 759 && ps.OverlayValues[759].Loc != scm.LocNone {
				d759 = ps.OverlayValues[759]
			}
			if len(ps.OverlayValues) > 760 && ps.OverlayValues[760].Loc != scm.LocNone {
				d760 = ps.OverlayValues[760]
			}
			if len(ps.OverlayValues) > 761 && ps.OverlayValues[761].Loc != scm.LocNone {
				d761 = ps.OverlayValues[761]
			}
			if len(ps.OverlayValues) > 762 && ps.OverlayValues[762].Loc != scm.LocNone {
				d762 = ps.OverlayValues[762]
			}
			if len(ps.OverlayValues) > 763 && ps.OverlayValues[763].Loc != scm.LocNone {
				d763 = ps.OverlayValues[763]
			}
			if len(ps.OverlayValues) > 764 && ps.OverlayValues[764].Loc != scm.LocNone {
				d764 = ps.OverlayValues[764]
			}
			if len(ps.OverlayValues) > 765 && ps.OverlayValues[765].Loc != scm.LocNone {
				d765 = ps.OverlayValues[765]
			}
			if len(ps.OverlayValues) > 766 && ps.OverlayValues[766].Loc != scm.LocNone {
				d766 = ps.OverlayValues[766]
			}
			if len(ps.OverlayValues) > 767 && ps.OverlayValues[767].Loc != scm.LocNone {
				d767 = ps.OverlayValues[767]
			}
			if len(ps.OverlayValues) > 768 && ps.OverlayValues[768].Loc != scm.LocNone {
				d768 = ps.OverlayValues[768]
			}
			if len(ps.OverlayValues) > 769 && ps.OverlayValues[769].Loc != scm.LocNone {
				d769 = ps.OverlayValues[769]
			}
			if len(ps.OverlayValues) > 770 && ps.OverlayValues[770].Loc != scm.LocNone {
				d770 = ps.OverlayValues[770]
			}
			if len(ps.OverlayValues) > 771 && ps.OverlayValues[771].Loc != scm.LocNone {
				d771 = ps.OverlayValues[771]
			}
			if len(ps.OverlayValues) > 772 && ps.OverlayValues[772].Loc != scm.LocNone {
				d772 = ps.OverlayValues[772]
			}
			if len(ps.OverlayValues) > 773 && ps.OverlayValues[773].Loc != scm.LocNone {
				d773 = ps.OverlayValues[773]
			}
			if len(ps.OverlayValues) > 774 && ps.OverlayValues[774].Loc != scm.LocNone {
				d774 = ps.OverlayValues[774]
			}
			if len(ps.OverlayValues) > 775 && ps.OverlayValues[775].Loc != scm.LocNone {
				d775 = ps.OverlayValues[775]
			}
			if len(ps.OverlayValues) > 776 && ps.OverlayValues[776].Loc != scm.LocNone {
				d776 = ps.OverlayValues[776]
			}
			if len(ps.OverlayValues) > 777 && ps.OverlayValues[777].Loc != scm.LocNone {
				d777 = ps.OverlayValues[777]
			}
			if len(ps.OverlayValues) > 778 && ps.OverlayValues[778].Loc != scm.LocNone {
				d778 = ps.OverlayValues[778]
			}
			if len(ps.OverlayValues) > 779 && ps.OverlayValues[779].Loc != scm.LocNone {
				d779 = ps.OverlayValues[779]
			}
			if len(ps.OverlayValues) > 780 && ps.OverlayValues[780].Loc != scm.LocNone {
				d780 = ps.OverlayValues[780]
			}
			if len(ps.OverlayValues) > 781 && ps.OverlayValues[781].Loc != scm.LocNone {
				d781 = ps.OverlayValues[781]
			}
			if len(ps.OverlayValues) > 782 && ps.OverlayValues[782].Loc != scm.LocNone {
				d782 = ps.OverlayValues[782]
			}
			if len(ps.OverlayValues) > 783 && ps.OverlayValues[783].Loc != scm.LocNone {
				d783 = ps.OverlayValues[783]
			}
			if len(ps.OverlayValues) > 784 && ps.OverlayValues[784].Loc != scm.LocNone {
				d784 = ps.OverlayValues[784]
			}
			if len(ps.OverlayValues) > 785 && ps.OverlayValues[785].Loc != scm.LocNone {
				d785 = ps.OverlayValues[785]
			}
			if len(ps.OverlayValues) > 786 && ps.OverlayValues[786].Loc != scm.LocNone {
				d786 = ps.OverlayValues[786]
			}
			if len(ps.OverlayValues) > 787 && ps.OverlayValues[787].Loc != scm.LocNone {
				d787 = ps.OverlayValues[787]
			}
			if len(ps.OverlayValues) > 1085 && ps.OverlayValues[1085].Loc != scm.LocNone {
				d1085 = ps.OverlayValues[1085]
			}
			if len(ps.OverlayValues) > 1086 && ps.OverlayValues[1086].Loc != scm.LocNone {
				d1086 = ps.OverlayValues[1086]
			}
			if len(ps.OverlayValues) > 1087 && ps.OverlayValues[1087].Loc != scm.LocNone {
				d1087 = ps.OverlayValues[1087]
			}
			if len(ps.OverlayValues) > 1088 && ps.OverlayValues[1088].Loc != scm.LocNone {
				d1088 = ps.OverlayValues[1088]
			}
			if len(ps.OverlayValues) > 1089 && ps.OverlayValues[1089].Loc != scm.LocNone {
				d1089 = ps.OverlayValues[1089]
			}
			if len(ps.OverlayValues) > 1090 && ps.OverlayValues[1090].Loc != scm.LocNone {
				d1090 = ps.OverlayValues[1090]
			}
			if len(ps.OverlayValues) > 1091 && ps.OverlayValues[1091].Loc != scm.LocNone {
				d1091 = ps.OverlayValues[1091]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d782)
			var d1092 scm.JITValueDesc
			if d782.Loc == scm.LocImm {
				d1092 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d782.Imm.Int() < 0)}
			} else {
				r298 := ctx.AllocRegExcept(d782.Reg)
				ctx.EmitCmpRegImm32(d782.Reg, 0)
				ctx.EmitSetcc(r298, scm.CcL)
				d1092 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r298}
				ctx.BindReg(r298, &d1092)
			}
			ctx.FreeDesc(&d782)
			d1093 = d1092
			ctx.EnsureDesc(&d1093)
			if d1093.Loc != scm.LocImm && d1093.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d1093.Loc == scm.LocImm {
				if d1093.Imm.Bool() {
			ps1094 := scm.PhiState{General: ps.General}
			ps1094.OverlayValues = make([]scm.JITValueDesc, 1094)
			ps1094.OverlayValues[0] = d0
			ps1094.OverlayValues[1] = d1
			ps1094.OverlayValues[2] = d2
			ps1094.OverlayValues[3] = d3
			ps1094.OverlayValues[4] = d4
			ps1094.OverlayValues[5] = d5
			ps1094.OverlayValues[6] = d6
			ps1094.OverlayValues[7] = d7
			ps1094.OverlayValues[8] = d8
			ps1094.OverlayValues[9] = d9
			ps1094.OverlayValues[10] = d10
			ps1094.OverlayValues[11] = d11
			ps1094.OverlayValues[12] = d12
			ps1094.OverlayValues[13] = d13
			ps1094.OverlayValues[14] = d14
			ps1094.OverlayValues[15] = d15
			ps1094.OverlayValues[16] = d16
			ps1094.OverlayValues[17] = d17
			ps1094.OverlayValues[18] = d18
			ps1094.OverlayValues[19] = d19
			ps1094.OverlayValues[20] = d20
			ps1094.OverlayValues[21] = d21
			ps1094.OverlayValues[22] = d22
			ps1094.OverlayValues[23] = d23
			ps1094.OverlayValues[24] = d24
			ps1094.OverlayValues[25] = d25
			ps1094.OverlayValues[26] = d26
			ps1094.OverlayValues[27] = d27
			ps1094.OverlayValues[28] = d28
			ps1094.OverlayValues[29] = d29
			ps1094.OverlayValues[30] = d30
			ps1094.OverlayValues[31] = d31
			ps1094.OverlayValues[32] = d32
			ps1094.OverlayValues[33] = d33
			ps1094.OverlayValues[34] = d34
			ps1094.OverlayValues[35] = d35
			ps1094.OverlayValues[36] = d36
			ps1094.OverlayValues[37] = d37
			ps1094.OverlayValues[38] = d38
			ps1094.OverlayValues[39] = d39
			ps1094.OverlayValues[40] = d40
			ps1094.OverlayValues[41] = d41
			ps1094.OverlayValues[42] = d42
			ps1094.OverlayValues[43] = d43
			ps1094.OverlayValues[44] = d44
			ps1094.OverlayValues[45] = d45
			ps1094.OverlayValues[46] = d46
			ps1094.OverlayValues[47] = d47
			ps1094.OverlayValues[48] = d48
			ps1094.OverlayValues[49] = d49
			ps1094.OverlayValues[50] = d50
			ps1094.OverlayValues[51] = d51
			ps1094.OverlayValues[52] = d52
			ps1094.OverlayValues[53] = d53
			ps1094.OverlayValues[54] = d54
			ps1094.OverlayValues[55] = d55
			ps1094.OverlayValues[56] = d56
			ps1094.OverlayValues[57] = d57
			ps1094.OverlayValues[58] = d58
			ps1094.OverlayValues[59] = d59
			ps1094.OverlayValues[60] = d60
			ps1094.OverlayValues[61] = d61
			ps1094.OverlayValues[62] = d62
			ps1094.OverlayValues[63] = d63
			ps1094.OverlayValues[64] = d64
			ps1094.OverlayValues[65] = d65
			ps1094.OverlayValues[66] = d66
			ps1094.OverlayValues[67] = d67
			ps1094.OverlayValues[68] = d68
			ps1094.OverlayValues[69] = d69
			ps1094.OverlayValues[70] = d70
			ps1094.OverlayValues[71] = d71
			ps1094.OverlayValues[72] = d72
			ps1094.OverlayValues[73] = d73
			ps1094.OverlayValues[74] = d74
			ps1094.OverlayValues[75] = d75
			ps1094.OverlayValues[76] = d76
			ps1094.OverlayValues[77] = d77
			ps1094.OverlayValues[78] = d78
			ps1094.OverlayValues[79] = d79
			ps1094.OverlayValues[80] = d80
			ps1094.OverlayValues[81] = d81
			ps1094.OverlayValues[82] = d82
			ps1094.OverlayValues[83] = d83
			ps1094.OverlayValues[84] = d84
			ps1094.OverlayValues[85] = d85
			ps1094.OverlayValues[86] = d86
			ps1094.OverlayValues[87] = d87
			ps1094.OverlayValues[88] = d88
			ps1094.OverlayValues[89] = d89
			ps1094.OverlayValues[90] = d90
			ps1094.OverlayValues[91] = d91
			ps1094.OverlayValues[92] = d92
			ps1094.OverlayValues[93] = d93
			ps1094.OverlayValues[94] = d94
			ps1094.OverlayValues[95] = d95
			ps1094.OverlayValues[96] = d96
			ps1094.OverlayValues[97] = d97
			ps1094.OverlayValues[98] = d98
			ps1094.OverlayValues[99] = d99
			ps1094.OverlayValues[100] = d100
			ps1094.OverlayValues[101] = d101
			ps1094.OverlayValues[102] = d102
			ps1094.OverlayValues[103] = d103
			ps1094.OverlayValues[104] = d104
			ps1094.OverlayValues[105] = d105
			ps1094.OverlayValues[106] = d106
			ps1094.OverlayValues[107] = d107
			ps1094.OverlayValues[108] = d108
			ps1094.OverlayValues[109] = d109
			ps1094.OverlayValues[110] = d110
			ps1094.OverlayValues[111] = d111
			ps1094.OverlayValues[112] = d112
			ps1094.OverlayValues[113] = d113
			ps1094.OverlayValues[114] = d114
			ps1094.OverlayValues[115] = d115
			ps1094.OverlayValues[116] = d116
			ps1094.OverlayValues[117] = d117
			ps1094.OverlayValues[118] = d118
			ps1094.OverlayValues[119] = d119
			ps1094.OverlayValues[120] = d120
			ps1094.OverlayValues[121] = d121
			ps1094.OverlayValues[122] = d122
			ps1094.OverlayValues[123] = d123
			ps1094.OverlayValues[124] = d124
			ps1094.OverlayValues[125] = d125
			ps1094.OverlayValues[126] = d126
			ps1094.OverlayValues[127] = d127
			ps1094.OverlayValues[128] = d128
			ps1094.OverlayValues[129] = d129
			ps1094.OverlayValues[130] = d130
			ps1094.OverlayValues[131] = d131
			ps1094.OverlayValues[132] = d132
			ps1094.OverlayValues[133] = d133
			ps1094.OverlayValues[134] = d134
			ps1094.OverlayValues[135] = d135
			ps1094.OverlayValues[136] = d136
			ps1094.OverlayValues[137] = d137
			ps1094.OverlayValues[138] = d138
			ps1094.OverlayValues[139] = d139
			ps1094.OverlayValues[140] = d140
			ps1094.OverlayValues[141] = d141
			ps1094.OverlayValues[142] = d142
			ps1094.OverlayValues[143] = d143
			ps1094.OverlayValues[144] = d144
			ps1094.OverlayValues[145] = d145
			ps1094.OverlayValues[146] = d146
			ps1094.OverlayValues[147] = d147
			ps1094.OverlayValues[148] = d148
			ps1094.OverlayValues[149] = d149
			ps1094.OverlayValues[150] = d150
			ps1094.OverlayValues[151] = d151
			ps1094.OverlayValues[152] = d152
			ps1094.OverlayValues[153] = d153
			ps1094.OverlayValues[154] = d154
			ps1094.OverlayValues[155] = d155
			ps1094.OverlayValues[156] = d156
			ps1094.OverlayValues[157] = d157
			ps1094.OverlayValues[158] = d158
			ps1094.OverlayValues[159] = d159
			ps1094.OverlayValues[160] = d160
			ps1094.OverlayValues[161] = d161
			ps1094.OverlayValues[162] = d162
			ps1094.OverlayValues[163] = d163
			ps1094.OverlayValues[164] = d164
			ps1094.OverlayValues[165] = d165
			ps1094.OverlayValues[166] = d166
			ps1094.OverlayValues[167] = d167
			ps1094.OverlayValues[168] = d168
			ps1094.OverlayValues[169] = d169
			ps1094.OverlayValues[170] = d170
			ps1094.OverlayValues[171] = d171
			ps1094.OverlayValues[172] = d172
			ps1094.OverlayValues[173] = d173
			ps1094.OverlayValues[174] = d174
			ps1094.OverlayValues[175] = d175
			ps1094.OverlayValues[176] = d176
			ps1094.OverlayValues[177] = d177
			ps1094.OverlayValues[178] = d178
			ps1094.OverlayValues[179] = d179
			ps1094.OverlayValues[180] = d180
			ps1094.OverlayValues[181] = d181
			ps1094.OverlayValues[182] = d182
			ps1094.OverlayValues[183] = d183
			ps1094.OverlayValues[184] = d184
			ps1094.OverlayValues[185] = d185
			ps1094.OverlayValues[186] = d186
			ps1094.OverlayValues[187] = d187
			ps1094.OverlayValues[188] = d188
			ps1094.OverlayValues[189] = d189
			ps1094.OverlayValues[190] = d190
			ps1094.OverlayValues[191] = d191
			ps1094.OverlayValues[192] = d192
			ps1094.OverlayValues[193] = d193
			ps1094.OverlayValues[194] = d194
			ps1094.OverlayValues[195] = d195
			ps1094.OverlayValues[196] = d196
			ps1094.OverlayValues[197] = d197
			ps1094.OverlayValues[198] = d198
			ps1094.OverlayValues[199] = d199
			ps1094.OverlayValues[200] = d200
			ps1094.OverlayValues[201] = d201
			ps1094.OverlayValues[202] = d202
			ps1094.OverlayValues[203] = d203
			ps1094.OverlayValues[204] = d204
			ps1094.OverlayValues[205] = d205
			ps1094.OverlayValues[206] = d206
			ps1094.OverlayValues[207] = d207
			ps1094.OverlayValues[208] = d208
			ps1094.OverlayValues[209] = d209
			ps1094.OverlayValues[210] = d210
			ps1094.OverlayValues[211] = d211
			ps1094.OverlayValues[212] = d212
			ps1094.OverlayValues[213] = d213
			ps1094.OverlayValues[214] = d214
			ps1094.OverlayValues[215] = d215
			ps1094.OverlayValues[216] = d216
			ps1094.OverlayValues[217] = d217
			ps1094.OverlayValues[218] = d218
			ps1094.OverlayValues[219] = d219
			ps1094.OverlayValues[220] = d220
			ps1094.OverlayValues[221] = d221
			ps1094.OverlayValues[222] = d222
			ps1094.OverlayValues[223] = d223
			ps1094.OverlayValues[224] = d224
			ps1094.OverlayValues[225] = d225
			ps1094.OverlayValues[226] = d226
			ps1094.OverlayValues[227] = d227
			ps1094.OverlayValues[228] = d228
			ps1094.OverlayValues[229] = d229
			ps1094.OverlayValues[230] = d230
			ps1094.OverlayValues[231] = d231
			ps1094.OverlayValues[232] = d232
			ps1094.OverlayValues[233] = d233
			ps1094.OverlayValues[234] = d234
			ps1094.OverlayValues[235] = d235
			ps1094.OverlayValues[236] = d236
			ps1094.OverlayValues[237] = d237
			ps1094.OverlayValues[238] = d238
			ps1094.OverlayValues[239] = d239
			ps1094.OverlayValues[240] = d240
			ps1094.OverlayValues[487] = d487
			ps1094.OverlayValues[488] = d488
			ps1094.OverlayValues[489] = d489
			ps1094.OverlayValues[490] = d490
			ps1094.OverlayValues[741] = d741
			ps1094.OverlayValues[742] = d742
			ps1094.OverlayValues[743] = d743
			ps1094.OverlayValues[744] = d744
			ps1094.OverlayValues[745] = d745
			ps1094.OverlayValues[746] = d746
			ps1094.OverlayValues[747] = d747
			ps1094.OverlayValues[748] = d748
			ps1094.OverlayValues[749] = d749
			ps1094.OverlayValues[750] = d750
			ps1094.OverlayValues[751] = d751
			ps1094.OverlayValues[752] = d752
			ps1094.OverlayValues[753] = d753
			ps1094.OverlayValues[754] = d754
			ps1094.OverlayValues[755] = d755
			ps1094.OverlayValues[756] = d756
			ps1094.OverlayValues[757] = d757
			ps1094.OverlayValues[758] = d758
			ps1094.OverlayValues[759] = d759
			ps1094.OverlayValues[760] = d760
			ps1094.OverlayValues[761] = d761
			ps1094.OverlayValues[762] = d762
			ps1094.OverlayValues[763] = d763
			ps1094.OverlayValues[764] = d764
			ps1094.OverlayValues[765] = d765
			ps1094.OverlayValues[766] = d766
			ps1094.OverlayValues[767] = d767
			ps1094.OverlayValues[768] = d768
			ps1094.OverlayValues[769] = d769
			ps1094.OverlayValues[770] = d770
			ps1094.OverlayValues[771] = d771
			ps1094.OverlayValues[772] = d772
			ps1094.OverlayValues[773] = d773
			ps1094.OverlayValues[774] = d774
			ps1094.OverlayValues[775] = d775
			ps1094.OverlayValues[776] = d776
			ps1094.OverlayValues[777] = d777
			ps1094.OverlayValues[778] = d778
			ps1094.OverlayValues[779] = d779
			ps1094.OverlayValues[780] = d780
			ps1094.OverlayValues[781] = d781
			ps1094.OverlayValues[782] = d782
			ps1094.OverlayValues[783] = d783
			ps1094.OverlayValues[784] = d784
			ps1094.OverlayValues[785] = d785
			ps1094.OverlayValues[786] = d786
			ps1094.OverlayValues[787] = d787
			ps1094.OverlayValues[1085] = d1085
			ps1094.OverlayValues[1086] = d1086
			ps1094.OverlayValues[1087] = d1087
			ps1094.OverlayValues[1088] = d1088
			ps1094.OverlayValues[1089] = d1089
			ps1094.OverlayValues[1090] = d1090
			ps1094.OverlayValues[1091] = d1091
			ps1094.OverlayValues[1092] = d1092
			ps1094.OverlayValues[1093] = d1093
					return bbs[5].RenderPS(ps1094)
				}
			ps1095 := scm.PhiState{General: ps.General}
			ps1095.OverlayValues = make([]scm.JITValueDesc, 1094)
			ps1095.OverlayValues[0] = d0
			ps1095.OverlayValues[1] = d1
			ps1095.OverlayValues[2] = d2
			ps1095.OverlayValues[3] = d3
			ps1095.OverlayValues[4] = d4
			ps1095.OverlayValues[5] = d5
			ps1095.OverlayValues[6] = d6
			ps1095.OverlayValues[7] = d7
			ps1095.OverlayValues[8] = d8
			ps1095.OverlayValues[9] = d9
			ps1095.OverlayValues[10] = d10
			ps1095.OverlayValues[11] = d11
			ps1095.OverlayValues[12] = d12
			ps1095.OverlayValues[13] = d13
			ps1095.OverlayValues[14] = d14
			ps1095.OverlayValues[15] = d15
			ps1095.OverlayValues[16] = d16
			ps1095.OverlayValues[17] = d17
			ps1095.OverlayValues[18] = d18
			ps1095.OverlayValues[19] = d19
			ps1095.OverlayValues[20] = d20
			ps1095.OverlayValues[21] = d21
			ps1095.OverlayValues[22] = d22
			ps1095.OverlayValues[23] = d23
			ps1095.OverlayValues[24] = d24
			ps1095.OverlayValues[25] = d25
			ps1095.OverlayValues[26] = d26
			ps1095.OverlayValues[27] = d27
			ps1095.OverlayValues[28] = d28
			ps1095.OverlayValues[29] = d29
			ps1095.OverlayValues[30] = d30
			ps1095.OverlayValues[31] = d31
			ps1095.OverlayValues[32] = d32
			ps1095.OverlayValues[33] = d33
			ps1095.OverlayValues[34] = d34
			ps1095.OverlayValues[35] = d35
			ps1095.OverlayValues[36] = d36
			ps1095.OverlayValues[37] = d37
			ps1095.OverlayValues[38] = d38
			ps1095.OverlayValues[39] = d39
			ps1095.OverlayValues[40] = d40
			ps1095.OverlayValues[41] = d41
			ps1095.OverlayValues[42] = d42
			ps1095.OverlayValues[43] = d43
			ps1095.OverlayValues[44] = d44
			ps1095.OverlayValues[45] = d45
			ps1095.OverlayValues[46] = d46
			ps1095.OverlayValues[47] = d47
			ps1095.OverlayValues[48] = d48
			ps1095.OverlayValues[49] = d49
			ps1095.OverlayValues[50] = d50
			ps1095.OverlayValues[51] = d51
			ps1095.OverlayValues[52] = d52
			ps1095.OverlayValues[53] = d53
			ps1095.OverlayValues[54] = d54
			ps1095.OverlayValues[55] = d55
			ps1095.OverlayValues[56] = d56
			ps1095.OverlayValues[57] = d57
			ps1095.OverlayValues[58] = d58
			ps1095.OverlayValues[59] = d59
			ps1095.OverlayValues[60] = d60
			ps1095.OverlayValues[61] = d61
			ps1095.OverlayValues[62] = d62
			ps1095.OverlayValues[63] = d63
			ps1095.OverlayValues[64] = d64
			ps1095.OverlayValues[65] = d65
			ps1095.OverlayValues[66] = d66
			ps1095.OverlayValues[67] = d67
			ps1095.OverlayValues[68] = d68
			ps1095.OverlayValues[69] = d69
			ps1095.OverlayValues[70] = d70
			ps1095.OverlayValues[71] = d71
			ps1095.OverlayValues[72] = d72
			ps1095.OverlayValues[73] = d73
			ps1095.OverlayValues[74] = d74
			ps1095.OverlayValues[75] = d75
			ps1095.OverlayValues[76] = d76
			ps1095.OverlayValues[77] = d77
			ps1095.OverlayValues[78] = d78
			ps1095.OverlayValues[79] = d79
			ps1095.OverlayValues[80] = d80
			ps1095.OverlayValues[81] = d81
			ps1095.OverlayValues[82] = d82
			ps1095.OverlayValues[83] = d83
			ps1095.OverlayValues[84] = d84
			ps1095.OverlayValues[85] = d85
			ps1095.OverlayValues[86] = d86
			ps1095.OverlayValues[87] = d87
			ps1095.OverlayValues[88] = d88
			ps1095.OverlayValues[89] = d89
			ps1095.OverlayValues[90] = d90
			ps1095.OverlayValues[91] = d91
			ps1095.OverlayValues[92] = d92
			ps1095.OverlayValues[93] = d93
			ps1095.OverlayValues[94] = d94
			ps1095.OverlayValues[95] = d95
			ps1095.OverlayValues[96] = d96
			ps1095.OverlayValues[97] = d97
			ps1095.OverlayValues[98] = d98
			ps1095.OverlayValues[99] = d99
			ps1095.OverlayValues[100] = d100
			ps1095.OverlayValues[101] = d101
			ps1095.OverlayValues[102] = d102
			ps1095.OverlayValues[103] = d103
			ps1095.OverlayValues[104] = d104
			ps1095.OverlayValues[105] = d105
			ps1095.OverlayValues[106] = d106
			ps1095.OverlayValues[107] = d107
			ps1095.OverlayValues[108] = d108
			ps1095.OverlayValues[109] = d109
			ps1095.OverlayValues[110] = d110
			ps1095.OverlayValues[111] = d111
			ps1095.OverlayValues[112] = d112
			ps1095.OverlayValues[113] = d113
			ps1095.OverlayValues[114] = d114
			ps1095.OverlayValues[115] = d115
			ps1095.OverlayValues[116] = d116
			ps1095.OverlayValues[117] = d117
			ps1095.OverlayValues[118] = d118
			ps1095.OverlayValues[119] = d119
			ps1095.OverlayValues[120] = d120
			ps1095.OverlayValues[121] = d121
			ps1095.OverlayValues[122] = d122
			ps1095.OverlayValues[123] = d123
			ps1095.OverlayValues[124] = d124
			ps1095.OverlayValues[125] = d125
			ps1095.OverlayValues[126] = d126
			ps1095.OverlayValues[127] = d127
			ps1095.OverlayValues[128] = d128
			ps1095.OverlayValues[129] = d129
			ps1095.OverlayValues[130] = d130
			ps1095.OverlayValues[131] = d131
			ps1095.OverlayValues[132] = d132
			ps1095.OverlayValues[133] = d133
			ps1095.OverlayValues[134] = d134
			ps1095.OverlayValues[135] = d135
			ps1095.OverlayValues[136] = d136
			ps1095.OverlayValues[137] = d137
			ps1095.OverlayValues[138] = d138
			ps1095.OverlayValues[139] = d139
			ps1095.OverlayValues[140] = d140
			ps1095.OverlayValues[141] = d141
			ps1095.OverlayValues[142] = d142
			ps1095.OverlayValues[143] = d143
			ps1095.OverlayValues[144] = d144
			ps1095.OverlayValues[145] = d145
			ps1095.OverlayValues[146] = d146
			ps1095.OverlayValues[147] = d147
			ps1095.OverlayValues[148] = d148
			ps1095.OverlayValues[149] = d149
			ps1095.OverlayValues[150] = d150
			ps1095.OverlayValues[151] = d151
			ps1095.OverlayValues[152] = d152
			ps1095.OverlayValues[153] = d153
			ps1095.OverlayValues[154] = d154
			ps1095.OverlayValues[155] = d155
			ps1095.OverlayValues[156] = d156
			ps1095.OverlayValues[157] = d157
			ps1095.OverlayValues[158] = d158
			ps1095.OverlayValues[159] = d159
			ps1095.OverlayValues[160] = d160
			ps1095.OverlayValues[161] = d161
			ps1095.OverlayValues[162] = d162
			ps1095.OverlayValues[163] = d163
			ps1095.OverlayValues[164] = d164
			ps1095.OverlayValues[165] = d165
			ps1095.OverlayValues[166] = d166
			ps1095.OverlayValues[167] = d167
			ps1095.OverlayValues[168] = d168
			ps1095.OverlayValues[169] = d169
			ps1095.OverlayValues[170] = d170
			ps1095.OverlayValues[171] = d171
			ps1095.OverlayValues[172] = d172
			ps1095.OverlayValues[173] = d173
			ps1095.OverlayValues[174] = d174
			ps1095.OverlayValues[175] = d175
			ps1095.OverlayValues[176] = d176
			ps1095.OverlayValues[177] = d177
			ps1095.OverlayValues[178] = d178
			ps1095.OverlayValues[179] = d179
			ps1095.OverlayValues[180] = d180
			ps1095.OverlayValues[181] = d181
			ps1095.OverlayValues[182] = d182
			ps1095.OverlayValues[183] = d183
			ps1095.OverlayValues[184] = d184
			ps1095.OverlayValues[185] = d185
			ps1095.OverlayValues[186] = d186
			ps1095.OverlayValues[187] = d187
			ps1095.OverlayValues[188] = d188
			ps1095.OverlayValues[189] = d189
			ps1095.OverlayValues[190] = d190
			ps1095.OverlayValues[191] = d191
			ps1095.OverlayValues[192] = d192
			ps1095.OverlayValues[193] = d193
			ps1095.OverlayValues[194] = d194
			ps1095.OverlayValues[195] = d195
			ps1095.OverlayValues[196] = d196
			ps1095.OverlayValues[197] = d197
			ps1095.OverlayValues[198] = d198
			ps1095.OverlayValues[199] = d199
			ps1095.OverlayValues[200] = d200
			ps1095.OverlayValues[201] = d201
			ps1095.OverlayValues[202] = d202
			ps1095.OverlayValues[203] = d203
			ps1095.OverlayValues[204] = d204
			ps1095.OverlayValues[205] = d205
			ps1095.OverlayValues[206] = d206
			ps1095.OverlayValues[207] = d207
			ps1095.OverlayValues[208] = d208
			ps1095.OverlayValues[209] = d209
			ps1095.OverlayValues[210] = d210
			ps1095.OverlayValues[211] = d211
			ps1095.OverlayValues[212] = d212
			ps1095.OverlayValues[213] = d213
			ps1095.OverlayValues[214] = d214
			ps1095.OverlayValues[215] = d215
			ps1095.OverlayValues[216] = d216
			ps1095.OverlayValues[217] = d217
			ps1095.OverlayValues[218] = d218
			ps1095.OverlayValues[219] = d219
			ps1095.OverlayValues[220] = d220
			ps1095.OverlayValues[221] = d221
			ps1095.OverlayValues[222] = d222
			ps1095.OverlayValues[223] = d223
			ps1095.OverlayValues[224] = d224
			ps1095.OverlayValues[225] = d225
			ps1095.OverlayValues[226] = d226
			ps1095.OverlayValues[227] = d227
			ps1095.OverlayValues[228] = d228
			ps1095.OverlayValues[229] = d229
			ps1095.OverlayValues[230] = d230
			ps1095.OverlayValues[231] = d231
			ps1095.OverlayValues[232] = d232
			ps1095.OverlayValues[233] = d233
			ps1095.OverlayValues[234] = d234
			ps1095.OverlayValues[235] = d235
			ps1095.OverlayValues[236] = d236
			ps1095.OverlayValues[237] = d237
			ps1095.OverlayValues[238] = d238
			ps1095.OverlayValues[239] = d239
			ps1095.OverlayValues[240] = d240
			ps1095.OverlayValues[487] = d487
			ps1095.OverlayValues[488] = d488
			ps1095.OverlayValues[489] = d489
			ps1095.OverlayValues[490] = d490
			ps1095.OverlayValues[741] = d741
			ps1095.OverlayValues[742] = d742
			ps1095.OverlayValues[743] = d743
			ps1095.OverlayValues[744] = d744
			ps1095.OverlayValues[745] = d745
			ps1095.OverlayValues[746] = d746
			ps1095.OverlayValues[747] = d747
			ps1095.OverlayValues[748] = d748
			ps1095.OverlayValues[749] = d749
			ps1095.OverlayValues[750] = d750
			ps1095.OverlayValues[751] = d751
			ps1095.OverlayValues[752] = d752
			ps1095.OverlayValues[753] = d753
			ps1095.OverlayValues[754] = d754
			ps1095.OverlayValues[755] = d755
			ps1095.OverlayValues[756] = d756
			ps1095.OverlayValues[757] = d757
			ps1095.OverlayValues[758] = d758
			ps1095.OverlayValues[759] = d759
			ps1095.OverlayValues[760] = d760
			ps1095.OverlayValues[761] = d761
			ps1095.OverlayValues[762] = d762
			ps1095.OverlayValues[763] = d763
			ps1095.OverlayValues[764] = d764
			ps1095.OverlayValues[765] = d765
			ps1095.OverlayValues[766] = d766
			ps1095.OverlayValues[767] = d767
			ps1095.OverlayValues[768] = d768
			ps1095.OverlayValues[769] = d769
			ps1095.OverlayValues[770] = d770
			ps1095.OverlayValues[771] = d771
			ps1095.OverlayValues[772] = d772
			ps1095.OverlayValues[773] = d773
			ps1095.OverlayValues[774] = d774
			ps1095.OverlayValues[775] = d775
			ps1095.OverlayValues[776] = d776
			ps1095.OverlayValues[777] = d777
			ps1095.OverlayValues[778] = d778
			ps1095.OverlayValues[779] = d779
			ps1095.OverlayValues[780] = d780
			ps1095.OverlayValues[781] = d781
			ps1095.OverlayValues[782] = d782
			ps1095.OverlayValues[783] = d783
			ps1095.OverlayValues[784] = d784
			ps1095.OverlayValues[785] = d785
			ps1095.OverlayValues[786] = d786
			ps1095.OverlayValues[787] = d787
			ps1095.OverlayValues[1085] = d1085
			ps1095.OverlayValues[1086] = d1086
			ps1095.OverlayValues[1087] = d1087
			ps1095.OverlayValues[1088] = d1088
			ps1095.OverlayValues[1089] = d1089
			ps1095.OverlayValues[1090] = d1090
			ps1095.OverlayValues[1091] = d1091
			ps1095.OverlayValues[1092] = d1092
			ps1095.OverlayValues[1093] = d1093
				return bbs[6].RenderPS(ps1095)
			}
			if !ps.General {
				ps.General = true
				return bbs[7].RenderPS(ps)
			}
			lbl82 := ctx.ReserveLabel()
			lbl83 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d1093.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl82)
			ctx.EmitJmp(lbl83)
			ctx.MarkLabel(lbl82)
			ctx.EmitJmp(lbl6)
			ctx.MarkLabel(lbl83)
			ctx.EmitJmp(lbl7)
			ps1096 := scm.PhiState{General: true}
			ps1096.OverlayValues = make([]scm.JITValueDesc, 1094)
			ps1096.OverlayValues[0] = d0
			ps1096.OverlayValues[1] = d1
			ps1096.OverlayValues[2] = d2
			ps1096.OverlayValues[3] = d3
			ps1096.OverlayValues[4] = d4
			ps1096.OverlayValues[5] = d5
			ps1096.OverlayValues[6] = d6
			ps1096.OverlayValues[7] = d7
			ps1096.OverlayValues[8] = d8
			ps1096.OverlayValues[9] = d9
			ps1096.OverlayValues[10] = d10
			ps1096.OverlayValues[11] = d11
			ps1096.OverlayValues[12] = d12
			ps1096.OverlayValues[13] = d13
			ps1096.OverlayValues[14] = d14
			ps1096.OverlayValues[15] = d15
			ps1096.OverlayValues[16] = d16
			ps1096.OverlayValues[17] = d17
			ps1096.OverlayValues[18] = d18
			ps1096.OverlayValues[19] = d19
			ps1096.OverlayValues[20] = d20
			ps1096.OverlayValues[21] = d21
			ps1096.OverlayValues[22] = d22
			ps1096.OverlayValues[23] = d23
			ps1096.OverlayValues[24] = d24
			ps1096.OverlayValues[25] = d25
			ps1096.OverlayValues[26] = d26
			ps1096.OverlayValues[27] = d27
			ps1096.OverlayValues[28] = d28
			ps1096.OverlayValues[29] = d29
			ps1096.OverlayValues[30] = d30
			ps1096.OverlayValues[31] = d31
			ps1096.OverlayValues[32] = d32
			ps1096.OverlayValues[33] = d33
			ps1096.OverlayValues[34] = d34
			ps1096.OverlayValues[35] = d35
			ps1096.OverlayValues[36] = d36
			ps1096.OverlayValues[37] = d37
			ps1096.OverlayValues[38] = d38
			ps1096.OverlayValues[39] = d39
			ps1096.OverlayValues[40] = d40
			ps1096.OverlayValues[41] = d41
			ps1096.OverlayValues[42] = d42
			ps1096.OverlayValues[43] = d43
			ps1096.OverlayValues[44] = d44
			ps1096.OverlayValues[45] = d45
			ps1096.OverlayValues[46] = d46
			ps1096.OverlayValues[47] = d47
			ps1096.OverlayValues[48] = d48
			ps1096.OverlayValues[49] = d49
			ps1096.OverlayValues[50] = d50
			ps1096.OverlayValues[51] = d51
			ps1096.OverlayValues[52] = d52
			ps1096.OverlayValues[53] = d53
			ps1096.OverlayValues[54] = d54
			ps1096.OverlayValues[55] = d55
			ps1096.OverlayValues[56] = d56
			ps1096.OverlayValues[57] = d57
			ps1096.OverlayValues[58] = d58
			ps1096.OverlayValues[59] = d59
			ps1096.OverlayValues[60] = d60
			ps1096.OverlayValues[61] = d61
			ps1096.OverlayValues[62] = d62
			ps1096.OverlayValues[63] = d63
			ps1096.OverlayValues[64] = d64
			ps1096.OverlayValues[65] = d65
			ps1096.OverlayValues[66] = d66
			ps1096.OverlayValues[67] = d67
			ps1096.OverlayValues[68] = d68
			ps1096.OverlayValues[69] = d69
			ps1096.OverlayValues[70] = d70
			ps1096.OverlayValues[71] = d71
			ps1096.OverlayValues[72] = d72
			ps1096.OverlayValues[73] = d73
			ps1096.OverlayValues[74] = d74
			ps1096.OverlayValues[75] = d75
			ps1096.OverlayValues[76] = d76
			ps1096.OverlayValues[77] = d77
			ps1096.OverlayValues[78] = d78
			ps1096.OverlayValues[79] = d79
			ps1096.OverlayValues[80] = d80
			ps1096.OverlayValues[81] = d81
			ps1096.OverlayValues[82] = d82
			ps1096.OverlayValues[83] = d83
			ps1096.OverlayValues[84] = d84
			ps1096.OverlayValues[85] = d85
			ps1096.OverlayValues[86] = d86
			ps1096.OverlayValues[87] = d87
			ps1096.OverlayValues[88] = d88
			ps1096.OverlayValues[89] = d89
			ps1096.OverlayValues[90] = d90
			ps1096.OverlayValues[91] = d91
			ps1096.OverlayValues[92] = d92
			ps1096.OverlayValues[93] = d93
			ps1096.OverlayValues[94] = d94
			ps1096.OverlayValues[95] = d95
			ps1096.OverlayValues[96] = d96
			ps1096.OverlayValues[97] = d97
			ps1096.OverlayValues[98] = d98
			ps1096.OverlayValues[99] = d99
			ps1096.OverlayValues[100] = d100
			ps1096.OverlayValues[101] = d101
			ps1096.OverlayValues[102] = d102
			ps1096.OverlayValues[103] = d103
			ps1096.OverlayValues[104] = d104
			ps1096.OverlayValues[105] = d105
			ps1096.OverlayValues[106] = d106
			ps1096.OverlayValues[107] = d107
			ps1096.OverlayValues[108] = d108
			ps1096.OverlayValues[109] = d109
			ps1096.OverlayValues[110] = d110
			ps1096.OverlayValues[111] = d111
			ps1096.OverlayValues[112] = d112
			ps1096.OverlayValues[113] = d113
			ps1096.OverlayValues[114] = d114
			ps1096.OverlayValues[115] = d115
			ps1096.OverlayValues[116] = d116
			ps1096.OverlayValues[117] = d117
			ps1096.OverlayValues[118] = d118
			ps1096.OverlayValues[119] = d119
			ps1096.OverlayValues[120] = d120
			ps1096.OverlayValues[121] = d121
			ps1096.OverlayValues[122] = d122
			ps1096.OverlayValues[123] = d123
			ps1096.OverlayValues[124] = d124
			ps1096.OverlayValues[125] = d125
			ps1096.OverlayValues[126] = d126
			ps1096.OverlayValues[127] = d127
			ps1096.OverlayValues[128] = d128
			ps1096.OverlayValues[129] = d129
			ps1096.OverlayValues[130] = d130
			ps1096.OverlayValues[131] = d131
			ps1096.OverlayValues[132] = d132
			ps1096.OverlayValues[133] = d133
			ps1096.OverlayValues[134] = d134
			ps1096.OverlayValues[135] = d135
			ps1096.OverlayValues[136] = d136
			ps1096.OverlayValues[137] = d137
			ps1096.OverlayValues[138] = d138
			ps1096.OverlayValues[139] = d139
			ps1096.OverlayValues[140] = d140
			ps1096.OverlayValues[141] = d141
			ps1096.OverlayValues[142] = d142
			ps1096.OverlayValues[143] = d143
			ps1096.OverlayValues[144] = d144
			ps1096.OverlayValues[145] = d145
			ps1096.OverlayValues[146] = d146
			ps1096.OverlayValues[147] = d147
			ps1096.OverlayValues[148] = d148
			ps1096.OverlayValues[149] = d149
			ps1096.OverlayValues[150] = d150
			ps1096.OverlayValues[151] = d151
			ps1096.OverlayValues[152] = d152
			ps1096.OverlayValues[153] = d153
			ps1096.OverlayValues[154] = d154
			ps1096.OverlayValues[155] = d155
			ps1096.OverlayValues[156] = d156
			ps1096.OverlayValues[157] = d157
			ps1096.OverlayValues[158] = d158
			ps1096.OverlayValues[159] = d159
			ps1096.OverlayValues[160] = d160
			ps1096.OverlayValues[161] = d161
			ps1096.OverlayValues[162] = d162
			ps1096.OverlayValues[163] = d163
			ps1096.OverlayValues[164] = d164
			ps1096.OverlayValues[165] = d165
			ps1096.OverlayValues[166] = d166
			ps1096.OverlayValues[167] = d167
			ps1096.OverlayValues[168] = d168
			ps1096.OverlayValues[169] = d169
			ps1096.OverlayValues[170] = d170
			ps1096.OverlayValues[171] = d171
			ps1096.OverlayValues[172] = d172
			ps1096.OverlayValues[173] = d173
			ps1096.OverlayValues[174] = d174
			ps1096.OverlayValues[175] = d175
			ps1096.OverlayValues[176] = d176
			ps1096.OverlayValues[177] = d177
			ps1096.OverlayValues[178] = d178
			ps1096.OverlayValues[179] = d179
			ps1096.OverlayValues[180] = d180
			ps1096.OverlayValues[181] = d181
			ps1096.OverlayValues[182] = d182
			ps1096.OverlayValues[183] = d183
			ps1096.OverlayValues[184] = d184
			ps1096.OverlayValues[185] = d185
			ps1096.OverlayValues[186] = d186
			ps1096.OverlayValues[187] = d187
			ps1096.OverlayValues[188] = d188
			ps1096.OverlayValues[189] = d189
			ps1096.OverlayValues[190] = d190
			ps1096.OverlayValues[191] = d191
			ps1096.OverlayValues[192] = d192
			ps1096.OverlayValues[193] = d193
			ps1096.OverlayValues[194] = d194
			ps1096.OverlayValues[195] = d195
			ps1096.OverlayValues[196] = d196
			ps1096.OverlayValues[197] = d197
			ps1096.OverlayValues[198] = d198
			ps1096.OverlayValues[199] = d199
			ps1096.OverlayValues[200] = d200
			ps1096.OverlayValues[201] = d201
			ps1096.OverlayValues[202] = d202
			ps1096.OverlayValues[203] = d203
			ps1096.OverlayValues[204] = d204
			ps1096.OverlayValues[205] = d205
			ps1096.OverlayValues[206] = d206
			ps1096.OverlayValues[207] = d207
			ps1096.OverlayValues[208] = d208
			ps1096.OverlayValues[209] = d209
			ps1096.OverlayValues[210] = d210
			ps1096.OverlayValues[211] = d211
			ps1096.OverlayValues[212] = d212
			ps1096.OverlayValues[213] = d213
			ps1096.OverlayValues[214] = d214
			ps1096.OverlayValues[215] = d215
			ps1096.OverlayValues[216] = d216
			ps1096.OverlayValues[217] = d217
			ps1096.OverlayValues[218] = d218
			ps1096.OverlayValues[219] = d219
			ps1096.OverlayValues[220] = d220
			ps1096.OverlayValues[221] = d221
			ps1096.OverlayValues[222] = d222
			ps1096.OverlayValues[223] = d223
			ps1096.OverlayValues[224] = d224
			ps1096.OverlayValues[225] = d225
			ps1096.OverlayValues[226] = d226
			ps1096.OverlayValues[227] = d227
			ps1096.OverlayValues[228] = d228
			ps1096.OverlayValues[229] = d229
			ps1096.OverlayValues[230] = d230
			ps1096.OverlayValues[231] = d231
			ps1096.OverlayValues[232] = d232
			ps1096.OverlayValues[233] = d233
			ps1096.OverlayValues[234] = d234
			ps1096.OverlayValues[235] = d235
			ps1096.OverlayValues[236] = d236
			ps1096.OverlayValues[237] = d237
			ps1096.OverlayValues[238] = d238
			ps1096.OverlayValues[239] = d239
			ps1096.OverlayValues[240] = d240
			ps1096.OverlayValues[487] = d487
			ps1096.OverlayValues[488] = d488
			ps1096.OverlayValues[489] = d489
			ps1096.OverlayValues[490] = d490
			ps1096.OverlayValues[741] = d741
			ps1096.OverlayValues[742] = d742
			ps1096.OverlayValues[743] = d743
			ps1096.OverlayValues[744] = d744
			ps1096.OverlayValues[745] = d745
			ps1096.OverlayValues[746] = d746
			ps1096.OverlayValues[747] = d747
			ps1096.OverlayValues[748] = d748
			ps1096.OverlayValues[749] = d749
			ps1096.OverlayValues[750] = d750
			ps1096.OverlayValues[751] = d751
			ps1096.OverlayValues[752] = d752
			ps1096.OverlayValues[753] = d753
			ps1096.OverlayValues[754] = d754
			ps1096.OverlayValues[755] = d755
			ps1096.OverlayValues[756] = d756
			ps1096.OverlayValues[757] = d757
			ps1096.OverlayValues[758] = d758
			ps1096.OverlayValues[759] = d759
			ps1096.OverlayValues[760] = d760
			ps1096.OverlayValues[761] = d761
			ps1096.OverlayValues[762] = d762
			ps1096.OverlayValues[763] = d763
			ps1096.OverlayValues[764] = d764
			ps1096.OverlayValues[765] = d765
			ps1096.OverlayValues[766] = d766
			ps1096.OverlayValues[767] = d767
			ps1096.OverlayValues[768] = d768
			ps1096.OverlayValues[769] = d769
			ps1096.OverlayValues[770] = d770
			ps1096.OverlayValues[771] = d771
			ps1096.OverlayValues[772] = d772
			ps1096.OverlayValues[773] = d773
			ps1096.OverlayValues[774] = d774
			ps1096.OverlayValues[775] = d775
			ps1096.OverlayValues[776] = d776
			ps1096.OverlayValues[777] = d777
			ps1096.OverlayValues[778] = d778
			ps1096.OverlayValues[779] = d779
			ps1096.OverlayValues[780] = d780
			ps1096.OverlayValues[781] = d781
			ps1096.OverlayValues[782] = d782
			ps1096.OverlayValues[783] = d783
			ps1096.OverlayValues[784] = d784
			ps1096.OverlayValues[785] = d785
			ps1096.OverlayValues[786] = d786
			ps1096.OverlayValues[787] = d787
			ps1096.OverlayValues[1085] = d1085
			ps1096.OverlayValues[1086] = d1086
			ps1096.OverlayValues[1087] = d1087
			ps1096.OverlayValues[1088] = d1088
			ps1096.OverlayValues[1089] = d1089
			ps1096.OverlayValues[1090] = d1090
			ps1096.OverlayValues[1091] = d1091
			ps1096.OverlayValues[1092] = d1092
			ps1096.OverlayValues[1093] = d1093
			ps1097 := scm.PhiState{General: true}
			ps1097.OverlayValues = make([]scm.JITValueDesc, 1094)
			ps1097.OverlayValues[0] = d0
			ps1097.OverlayValues[1] = d1
			ps1097.OverlayValues[2] = d2
			ps1097.OverlayValues[3] = d3
			ps1097.OverlayValues[4] = d4
			ps1097.OverlayValues[5] = d5
			ps1097.OverlayValues[6] = d6
			ps1097.OverlayValues[7] = d7
			ps1097.OverlayValues[8] = d8
			ps1097.OverlayValues[9] = d9
			ps1097.OverlayValues[10] = d10
			ps1097.OverlayValues[11] = d11
			ps1097.OverlayValues[12] = d12
			ps1097.OverlayValues[13] = d13
			ps1097.OverlayValues[14] = d14
			ps1097.OverlayValues[15] = d15
			ps1097.OverlayValues[16] = d16
			ps1097.OverlayValues[17] = d17
			ps1097.OverlayValues[18] = d18
			ps1097.OverlayValues[19] = d19
			ps1097.OverlayValues[20] = d20
			ps1097.OverlayValues[21] = d21
			ps1097.OverlayValues[22] = d22
			ps1097.OverlayValues[23] = d23
			ps1097.OverlayValues[24] = d24
			ps1097.OverlayValues[25] = d25
			ps1097.OverlayValues[26] = d26
			ps1097.OverlayValues[27] = d27
			ps1097.OverlayValues[28] = d28
			ps1097.OverlayValues[29] = d29
			ps1097.OverlayValues[30] = d30
			ps1097.OverlayValues[31] = d31
			ps1097.OverlayValues[32] = d32
			ps1097.OverlayValues[33] = d33
			ps1097.OverlayValues[34] = d34
			ps1097.OverlayValues[35] = d35
			ps1097.OverlayValues[36] = d36
			ps1097.OverlayValues[37] = d37
			ps1097.OverlayValues[38] = d38
			ps1097.OverlayValues[39] = d39
			ps1097.OverlayValues[40] = d40
			ps1097.OverlayValues[41] = d41
			ps1097.OverlayValues[42] = d42
			ps1097.OverlayValues[43] = d43
			ps1097.OverlayValues[44] = d44
			ps1097.OverlayValues[45] = d45
			ps1097.OverlayValues[46] = d46
			ps1097.OverlayValues[47] = d47
			ps1097.OverlayValues[48] = d48
			ps1097.OverlayValues[49] = d49
			ps1097.OverlayValues[50] = d50
			ps1097.OverlayValues[51] = d51
			ps1097.OverlayValues[52] = d52
			ps1097.OverlayValues[53] = d53
			ps1097.OverlayValues[54] = d54
			ps1097.OverlayValues[55] = d55
			ps1097.OverlayValues[56] = d56
			ps1097.OverlayValues[57] = d57
			ps1097.OverlayValues[58] = d58
			ps1097.OverlayValues[59] = d59
			ps1097.OverlayValues[60] = d60
			ps1097.OverlayValues[61] = d61
			ps1097.OverlayValues[62] = d62
			ps1097.OverlayValues[63] = d63
			ps1097.OverlayValues[64] = d64
			ps1097.OverlayValues[65] = d65
			ps1097.OverlayValues[66] = d66
			ps1097.OverlayValues[67] = d67
			ps1097.OverlayValues[68] = d68
			ps1097.OverlayValues[69] = d69
			ps1097.OverlayValues[70] = d70
			ps1097.OverlayValues[71] = d71
			ps1097.OverlayValues[72] = d72
			ps1097.OverlayValues[73] = d73
			ps1097.OverlayValues[74] = d74
			ps1097.OverlayValues[75] = d75
			ps1097.OverlayValues[76] = d76
			ps1097.OverlayValues[77] = d77
			ps1097.OverlayValues[78] = d78
			ps1097.OverlayValues[79] = d79
			ps1097.OverlayValues[80] = d80
			ps1097.OverlayValues[81] = d81
			ps1097.OverlayValues[82] = d82
			ps1097.OverlayValues[83] = d83
			ps1097.OverlayValues[84] = d84
			ps1097.OverlayValues[85] = d85
			ps1097.OverlayValues[86] = d86
			ps1097.OverlayValues[87] = d87
			ps1097.OverlayValues[88] = d88
			ps1097.OverlayValues[89] = d89
			ps1097.OverlayValues[90] = d90
			ps1097.OverlayValues[91] = d91
			ps1097.OverlayValues[92] = d92
			ps1097.OverlayValues[93] = d93
			ps1097.OverlayValues[94] = d94
			ps1097.OverlayValues[95] = d95
			ps1097.OverlayValues[96] = d96
			ps1097.OverlayValues[97] = d97
			ps1097.OverlayValues[98] = d98
			ps1097.OverlayValues[99] = d99
			ps1097.OverlayValues[100] = d100
			ps1097.OverlayValues[101] = d101
			ps1097.OverlayValues[102] = d102
			ps1097.OverlayValues[103] = d103
			ps1097.OverlayValues[104] = d104
			ps1097.OverlayValues[105] = d105
			ps1097.OverlayValues[106] = d106
			ps1097.OverlayValues[107] = d107
			ps1097.OverlayValues[108] = d108
			ps1097.OverlayValues[109] = d109
			ps1097.OverlayValues[110] = d110
			ps1097.OverlayValues[111] = d111
			ps1097.OverlayValues[112] = d112
			ps1097.OverlayValues[113] = d113
			ps1097.OverlayValues[114] = d114
			ps1097.OverlayValues[115] = d115
			ps1097.OverlayValues[116] = d116
			ps1097.OverlayValues[117] = d117
			ps1097.OverlayValues[118] = d118
			ps1097.OverlayValues[119] = d119
			ps1097.OverlayValues[120] = d120
			ps1097.OverlayValues[121] = d121
			ps1097.OverlayValues[122] = d122
			ps1097.OverlayValues[123] = d123
			ps1097.OverlayValues[124] = d124
			ps1097.OverlayValues[125] = d125
			ps1097.OverlayValues[126] = d126
			ps1097.OverlayValues[127] = d127
			ps1097.OverlayValues[128] = d128
			ps1097.OverlayValues[129] = d129
			ps1097.OverlayValues[130] = d130
			ps1097.OverlayValues[131] = d131
			ps1097.OverlayValues[132] = d132
			ps1097.OverlayValues[133] = d133
			ps1097.OverlayValues[134] = d134
			ps1097.OverlayValues[135] = d135
			ps1097.OverlayValues[136] = d136
			ps1097.OverlayValues[137] = d137
			ps1097.OverlayValues[138] = d138
			ps1097.OverlayValues[139] = d139
			ps1097.OverlayValues[140] = d140
			ps1097.OverlayValues[141] = d141
			ps1097.OverlayValues[142] = d142
			ps1097.OverlayValues[143] = d143
			ps1097.OverlayValues[144] = d144
			ps1097.OverlayValues[145] = d145
			ps1097.OverlayValues[146] = d146
			ps1097.OverlayValues[147] = d147
			ps1097.OverlayValues[148] = d148
			ps1097.OverlayValues[149] = d149
			ps1097.OverlayValues[150] = d150
			ps1097.OverlayValues[151] = d151
			ps1097.OverlayValues[152] = d152
			ps1097.OverlayValues[153] = d153
			ps1097.OverlayValues[154] = d154
			ps1097.OverlayValues[155] = d155
			ps1097.OverlayValues[156] = d156
			ps1097.OverlayValues[157] = d157
			ps1097.OverlayValues[158] = d158
			ps1097.OverlayValues[159] = d159
			ps1097.OverlayValues[160] = d160
			ps1097.OverlayValues[161] = d161
			ps1097.OverlayValues[162] = d162
			ps1097.OverlayValues[163] = d163
			ps1097.OverlayValues[164] = d164
			ps1097.OverlayValues[165] = d165
			ps1097.OverlayValues[166] = d166
			ps1097.OverlayValues[167] = d167
			ps1097.OverlayValues[168] = d168
			ps1097.OverlayValues[169] = d169
			ps1097.OverlayValues[170] = d170
			ps1097.OverlayValues[171] = d171
			ps1097.OverlayValues[172] = d172
			ps1097.OverlayValues[173] = d173
			ps1097.OverlayValues[174] = d174
			ps1097.OverlayValues[175] = d175
			ps1097.OverlayValues[176] = d176
			ps1097.OverlayValues[177] = d177
			ps1097.OverlayValues[178] = d178
			ps1097.OverlayValues[179] = d179
			ps1097.OverlayValues[180] = d180
			ps1097.OverlayValues[181] = d181
			ps1097.OverlayValues[182] = d182
			ps1097.OverlayValues[183] = d183
			ps1097.OverlayValues[184] = d184
			ps1097.OverlayValues[185] = d185
			ps1097.OverlayValues[186] = d186
			ps1097.OverlayValues[187] = d187
			ps1097.OverlayValues[188] = d188
			ps1097.OverlayValues[189] = d189
			ps1097.OverlayValues[190] = d190
			ps1097.OverlayValues[191] = d191
			ps1097.OverlayValues[192] = d192
			ps1097.OverlayValues[193] = d193
			ps1097.OverlayValues[194] = d194
			ps1097.OverlayValues[195] = d195
			ps1097.OverlayValues[196] = d196
			ps1097.OverlayValues[197] = d197
			ps1097.OverlayValues[198] = d198
			ps1097.OverlayValues[199] = d199
			ps1097.OverlayValues[200] = d200
			ps1097.OverlayValues[201] = d201
			ps1097.OverlayValues[202] = d202
			ps1097.OverlayValues[203] = d203
			ps1097.OverlayValues[204] = d204
			ps1097.OverlayValues[205] = d205
			ps1097.OverlayValues[206] = d206
			ps1097.OverlayValues[207] = d207
			ps1097.OverlayValues[208] = d208
			ps1097.OverlayValues[209] = d209
			ps1097.OverlayValues[210] = d210
			ps1097.OverlayValues[211] = d211
			ps1097.OverlayValues[212] = d212
			ps1097.OverlayValues[213] = d213
			ps1097.OverlayValues[214] = d214
			ps1097.OverlayValues[215] = d215
			ps1097.OverlayValues[216] = d216
			ps1097.OverlayValues[217] = d217
			ps1097.OverlayValues[218] = d218
			ps1097.OverlayValues[219] = d219
			ps1097.OverlayValues[220] = d220
			ps1097.OverlayValues[221] = d221
			ps1097.OverlayValues[222] = d222
			ps1097.OverlayValues[223] = d223
			ps1097.OverlayValues[224] = d224
			ps1097.OverlayValues[225] = d225
			ps1097.OverlayValues[226] = d226
			ps1097.OverlayValues[227] = d227
			ps1097.OverlayValues[228] = d228
			ps1097.OverlayValues[229] = d229
			ps1097.OverlayValues[230] = d230
			ps1097.OverlayValues[231] = d231
			ps1097.OverlayValues[232] = d232
			ps1097.OverlayValues[233] = d233
			ps1097.OverlayValues[234] = d234
			ps1097.OverlayValues[235] = d235
			ps1097.OverlayValues[236] = d236
			ps1097.OverlayValues[237] = d237
			ps1097.OverlayValues[238] = d238
			ps1097.OverlayValues[239] = d239
			ps1097.OverlayValues[240] = d240
			ps1097.OverlayValues[487] = d487
			ps1097.OverlayValues[488] = d488
			ps1097.OverlayValues[489] = d489
			ps1097.OverlayValues[490] = d490
			ps1097.OverlayValues[741] = d741
			ps1097.OverlayValues[742] = d742
			ps1097.OverlayValues[743] = d743
			ps1097.OverlayValues[744] = d744
			ps1097.OverlayValues[745] = d745
			ps1097.OverlayValues[746] = d746
			ps1097.OverlayValues[747] = d747
			ps1097.OverlayValues[748] = d748
			ps1097.OverlayValues[749] = d749
			ps1097.OverlayValues[750] = d750
			ps1097.OverlayValues[751] = d751
			ps1097.OverlayValues[752] = d752
			ps1097.OverlayValues[753] = d753
			ps1097.OverlayValues[754] = d754
			ps1097.OverlayValues[755] = d755
			ps1097.OverlayValues[756] = d756
			ps1097.OverlayValues[757] = d757
			ps1097.OverlayValues[758] = d758
			ps1097.OverlayValues[759] = d759
			ps1097.OverlayValues[760] = d760
			ps1097.OverlayValues[761] = d761
			ps1097.OverlayValues[762] = d762
			ps1097.OverlayValues[763] = d763
			ps1097.OverlayValues[764] = d764
			ps1097.OverlayValues[765] = d765
			ps1097.OverlayValues[766] = d766
			ps1097.OverlayValues[767] = d767
			ps1097.OverlayValues[768] = d768
			ps1097.OverlayValues[769] = d769
			ps1097.OverlayValues[770] = d770
			ps1097.OverlayValues[771] = d771
			ps1097.OverlayValues[772] = d772
			ps1097.OverlayValues[773] = d773
			ps1097.OverlayValues[774] = d774
			ps1097.OverlayValues[775] = d775
			ps1097.OverlayValues[776] = d776
			ps1097.OverlayValues[777] = d777
			ps1097.OverlayValues[778] = d778
			ps1097.OverlayValues[779] = d779
			ps1097.OverlayValues[780] = d780
			ps1097.OverlayValues[781] = d781
			ps1097.OverlayValues[782] = d782
			ps1097.OverlayValues[783] = d783
			ps1097.OverlayValues[784] = d784
			ps1097.OverlayValues[785] = d785
			ps1097.OverlayValues[786] = d786
			ps1097.OverlayValues[787] = d787
			ps1097.OverlayValues[1085] = d1085
			ps1097.OverlayValues[1086] = d1086
			ps1097.OverlayValues[1087] = d1087
			ps1097.OverlayValues[1088] = d1088
			ps1097.OverlayValues[1089] = d1089
			ps1097.OverlayValues[1090] = d1090
			ps1097.OverlayValues[1091] = d1091
			ps1097.OverlayValues[1092] = d1092
			ps1097.OverlayValues[1093] = d1093
			snap1098 := d0
			snap1099 := d1
			snap1100 := d2
			snap1101 := d3
			snap1102 := d4
			snap1103 := d5
			snap1104 := d6
			snap1105 := d7
			snap1106 := d8
			snap1107 := d9
			snap1108 := d10
			snap1109 := d11
			snap1110 := d12
			snap1111 := d13
			snap1112 := d14
			snap1113 := d15
			snap1114 := d16
			snap1115 := d17
			snap1116 := d18
			snap1117 := d19
			snap1118 := d20
			snap1119 := d21
			snap1120 := d22
			snap1121 := d23
			snap1122 := d24
			snap1123 := d25
			snap1124 := d26
			snap1125 := d27
			snap1126 := d28
			snap1127 := d29
			snap1128 := d30
			snap1129 := d31
			snap1130 := d32
			snap1131 := d33
			snap1132 := d34
			snap1133 := d35
			snap1134 := d36
			snap1135 := d37
			snap1136 := d38
			snap1137 := d39
			snap1138 := d40
			snap1139 := d41
			snap1140 := d42
			snap1141 := d43
			snap1142 := d44
			snap1143 := d45
			snap1144 := d46
			snap1145 := d47
			snap1146 := d48
			snap1147 := d49
			snap1148 := d50
			snap1149 := d51
			snap1150 := d52
			snap1151 := d53
			snap1152 := d54
			snap1153 := d55
			snap1154 := d56
			snap1155 := d57
			snap1156 := d58
			snap1157 := d59
			snap1158 := d60
			snap1159 := d61
			snap1160 := d62
			snap1161 := d63
			snap1162 := d64
			snap1163 := d65
			snap1164 := d66
			snap1165 := d67
			snap1166 := d68
			snap1167 := d69
			snap1168 := d70
			snap1169 := d71
			snap1170 := d72
			snap1171 := d73
			snap1172 := d74
			snap1173 := d75
			snap1174 := d76
			snap1175 := d77
			snap1176 := d78
			snap1177 := d79
			snap1178 := d80
			snap1179 := d81
			snap1180 := d82
			snap1181 := d83
			snap1182 := d84
			snap1183 := d85
			snap1184 := d86
			snap1185 := d87
			snap1186 := d88
			snap1187 := d89
			snap1188 := d90
			snap1189 := d91
			snap1190 := d92
			snap1191 := d93
			snap1192 := d94
			snap1193 := d95
			snap1194 := d96
			snap1195 := d97
			snap1196 := d98
			snap1197 := d99
			snap1198 := d100
			snap1199 := d101
			snap1200 := d102
			snap1201 := d103
			snap1202 := d104
			snap1203 := d105
			snap1204 := d106
			snap1205 := d107
			snap1206 := d108
			snap1207 := d109
			snap1208 := d110
			snap1209 := d111
			snap1210 := d112
			snap1211 := d113
			snap1212 := d114
			snap1213 := d115
			snap1214 := d116
			snap1215 := d117
			snap1216 := d118
			snap1217 := d119
			snap1218 := d120
			snap1219 := d121
			snap1220 := d122
			snap1221 := d123
			snap1222 := d124
			snap1223 := d125
			snap1224 := d126
			snap1225 := d127
			snap1226 := d128
			snap1227 := d129
			snap1228 := d130
			snap1229 := d131
			snap1230 := d132
			snap1231 := d133
			snap1232 := d134
			snap1233 := d135
			snap1234 := d136
			snap1235 := d137
			snap1236 := d138
			snap1237 := d139
			snap1238 := d140
			snap1239 := d141
			snap1240 := d142
			snap1241 := d143
			snap1242 := d144
			snap1243 := d145
			snap1244 := d146
			snap1245 := d147
			snap1246 := d148
			snap1247 := d149
			snap1248 := d150
			snap1249 := d151
			snap1250 := d152
			snap1251 := d153
			snap1252 := d154
			snap1253 := d155
			snap1254 := d156
			snap1255 := d157
			snap1256 := d158
			snap1257 := d159
			snap1258 := d160
			snap1259 := d161
			snap1260 := d162
			snap1261 := d163
			snap1262 := d164
			snap1263 := d165
			snap1264 := d166
			snap1265 := d167
			snap1266 := d168
			snap1267 := d169
			snap1268 := d170
			snap1269 := d171
			snap1270 := d172
			snap1271 := d173
			snap1272 := d174
			snap1273 := d175
			snap1274 := d176
			snap1275 := d177
			snap1276 := d178
			snap1277 := d179
			snap1278 := d180
			snap1279 := d181
			snap1280 := d182
			snap1281 := d183
			snap1282 := d184
			snap1283 := d185
			snap1284 := d186
			snap1285 := d187
			snap1286 := d188
			snap1287 := d189
			snap1288 := d190
			snap1289 := d191
			snap1290 := d192
			snap1291 := d193
			snap1292 := d194
			snap1293 := d195
			snap1294 := d196
			snap1295 := d197
			snap1296 := d198
			snap1297 := d199
			snap1298 := d200
			snap1299 := d201
			snap1300 := d202
			snap1301 := d203
			snap1302 := d204
			snap1303 := d205
			snap1304 := d206
			snap1305 := d207
			snap1306 := d208
			snap1307 := d209
			snap1308 := d210
			snap1309 := d211
			snap1310 := d212
			snap1311 := d213
			snap1312 := d214
			snap1313 := d215
			snap1314 := d216
			snap1315 := d217
			snap1316 := d218
			snap1317 := d219
			snap1318 := d220
			snap1319 := d221
			snap1320 := d222
			snap1321 := d223
			snap1322 := d224
			snap1323 := d225
			snap1324 := d226
			snap1325 := d227
			snap1326 := d228
			snap1327 := d229
			snap1328 := d230
			snap1329 := d231
			snap1330 := d232
			snap1331 := d233
			snap1332 := d234
			snap1333 := d235
			snap1334 := d236
			snap1335 := d237
			snap1336 := d238
			snap1337 := d239
			snap1338 := d240
			snap1339 := d487
			snap1340 := d488
			snap1341 := d489
			snap1342 := d490
			snap1343 := d741
			snap1344 := d742
			snap1345 := d743
			snap1346 := d744
			snap1347 := d745
			snap1348 := d746
			snap1349 := d747
			snap1350 := d748
			snap1351 := d749
			snap1352 := d750
			snap1353 := d751
			snap1354 := d752
			snap1355 := d753
			snap1356 := d754
			snap1357 := d755
			snap1358 := d756
			snap1359 := d757
			snap1360 := d758
			snap1361 := d759
			snap1362 := d760
			snap1363 := d761
			snap1364 := d762
			snap1365 := d763
			snap1366 := d764
			snap1367 := d765
			snap1368 := d766
			snap1369 := d767
			snap1370 := d768
			snap1371 := d769
			snap1372 := d770
			snap1373 := d771
			snap1374 := d772
			snap1375 := d773
			snap1376 := d774
			snap1377 := d775
			snap1378 := d776
			snap1379 := d777
			snap1380 := d778
			snap1381 := d779
			snap1382 := d780
			snap1383 := d781
			snap1384 := d782
			snap1385 := d783
			snap1386 := d784
			snap1387 := d785
			snap1388 := d786
			snap1389 := d787
			snap1390 := d1085
			snap1391 := d1086
			snap1392 := d1087
			snap1393 := d1088
			snap1394 := d1089
			snap1395 := d1090
			snap1396 := d1091
			snap1397 := d1092
			snap1398 := d1093
			alloc1399 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps1097)
			}
			ctx.RestoreAllocState(alloc1399)
			d0 = snap1098
			d1 = snap1099
			d2 = snap1100
			d3 = snap1101
			d4 = snap1102
			d5 = snap1103
			d6 = snap1104
			d7 = snap1105
			d8 = snap1106
			d9 = snap1107
			d10 = snap1108
			d11 = snap1109
			d12 = snap1110
			d13 = snap1111
			d14 = snap1112
			d15 = snap1113
			d16 = snap1114
			d17 = snap1115
			d18 = snap1116
			d19 = snap1117
			d20 = snap1118
			d21 = snap1119
			d22 = snap1120
			d23 = snap1121
			d24 = snap1122
			d25 = snap1123
			d26 = snap1124
			d27 = snap1125
			d28 = snap1126
			d29 = snap1127
			d30 = snap1128
			d31 = snap1129
			d32 = snap1130
			d33 = snap1131
			d34 = snap1132
			d35 = snap1133
			d36 = snap1134
			d37 = snap1135
			d38 = snap1136
			d39 = snap1137
			d40 = snap1138
			d41 = snap1139
			d42 = snap1140
			d43 = snap1141
			d44 = snap1142
			d45 = snap1143
			d46 = snap1144
			d47 = snap1145
			d48 = snap1146
			d49 = snap1147
			d50 = snap1148
			d51 = snap1149
			d52 = snap1150
			d53 = snap1151
			d54 = snap1152
			d55 = snap1153
			d56 = snap1154
			d57 = snap1155
			d58 = snap1156
			d59 = snap1157
			d60 = snap1158
			d61 = snap1159
			d62 = snap1160
			d63 = snap1161
			d64 = snap1162
			d65 = snap1163
			d66 = snap1164
			d67 = snap1165
			d68 = snap1166
			d69 = snap1167
			d70 = snap1168
			d71 = snap1169
			d72 = snap1170
			d73 = snap1171
			d74 = snap1172
			d75 = snap1173
			d76 = snap1174
			d77 = snap1175
			d78 = snap1176
			d79 = snap1177
			d80 = snap1178
			d81 = snap1179
			d82 = snap1180
			d83 = snap1181
			d84 = snap1182
			d85 = snap1183
			d86 = snap1184
			d87 = snap1185
			d88 = snap1186
			d89 = snap1187
			d90 = snap1188
			d91 = snap1189
			d92 = snap1190
			d93 = snap1191
			d94 = snap1192
			d95 = snap1193
			d96 = snap1194
			d97 = snap1195
			d98 = snap1196
			d99 = snap1197
			d100 = snap1198
			d101 = snap1199
			d102 = snap1200
			d103 = snap1201
			d104 = snap1202
			d105 = snap1203
			d106 = snap1204
			d107 = snap1205
			d108 = snap1206
			d109 = snap1207
			d110 = snap1208
			d111 = snap1209
			d112 = snap1210
			d113 = snap1211
			d114 = snap1212
			d115 = snap1213
			d116 = snap1214
			d117 = snap1215
			d118 = snap1216
			d119 = snap1217
			d120 = snap1218
			d121 = snap1219
			d122 = snap1220
			d123 = snap1221
			d124 = snap1222
			d125 = snap1223
			d126 = snap1224
			d127 = snap1225
			d128 = snap1226
			d129 = snap1227
			d130 = snap1228
			d131 = snap1229
			d132 = snap1230
			d133 = snap1231
			d134 = snap1232
			d135 = snap1233
			d136 = snap1234
			d137 = snap1235
			d138 = snap1236
			d139 = snap1237
			d140 = snap1238
			d141 = snap1239
			d142 = snap1240
			d143 = snap1241
			d144 = snap1242
			d145 = snap1243
			d146 = snap1244
			d147 = snap1245
			d148 = snap1246
			d149 = snap1247
			d150 = snap1248
			d151 = snap1249
			d152 = snap1250
			d153 = snap1251
			d154 = snap1252
			d155 = snap1253
			d156 = snap1254
			d157 = snap1255
			d158 = snap1256
			d159 = snap1257
			d160 = snap1258
			d161 = snap1259
			d162 = snap1260
			d163 = snap1261
			d164 = snap1262
			d165 = snap1263
			d166 = snap1264
			d167 = snap1265
			d168 = snap1266
			d169 = snap1267
			d170 = snap1268
			d171 = snap1269
			d172 = snap1270
			d173 = snap1271
			d174 = snap1272
			d175 = snap1273
			d176 = snap1274
			d177 = snap1275
			d178 = snap1276
			d179 = snap1277
			d180 = snap1278
			d181 = snap1279
			d182 = snap1280
			d183 = snap1281
			d184 = snap1282
			d185 = snap1283
			d186 = snap1284
			d187 = snap1285
			d188 = snap1286
			d189 = snap1287
			d190 = snap1288
			d191 = snap1289
			d192 = snap1290
			d193 = snap1291
			d194 = snap1292
			d195 = snap1293
			d196 = snap1294
			d197 = snap1295
			d198 = snap1296
			d199 = snap1297
			d200 = snap1298
			d201 = snap1299
			d202 = snap1300
			d203 = snap1301
			d204 = snap1302
			d205 = snap1303
			d206 = snap1304
			d207 = snap1305
			d208 = snap1306
			d209 = snap1307
			d210 = snap1308
			d211 = snap1309
			d212 = snap1310
			d213 = snap1311
			d214 = snap1312
			d215 = snap1313
			d216 = snap1314
			d217 = snap1315
			d218 = snap1316
			d219 = snap1317
			d220 = snap1318
			d221 = snap1319
			d222 = snap1320
			d223 = snap1321
			d224 = snap1322
			d225 = snap1323
			d226 = snap1324
			d227 = snap1325
			d228 = snap1326
			d229 = snap1327
			d230 = snap1328
			d231 = snap1329
			d232 = snap1330
			d233 = snap1331
			d234 = snap1332
			d235 = snap1333
			d236 = snap1334
			d237 = snap1335
			d238 = snap1336
			d239 = snap1337
			d240 = snap1338
			d487 = snap1339
			d488 = snap1340
			d489 = snap1341
			d490 = snap1342
			d741 = snap1343
			d742 = snap1344
			d743 = snap1345
			d744 = snap1346
			d745 = snap1347
			d746 = snap1348
			d747 = snap1349
			d748 = snap1350
			d749 = snap1351
			d750 = snap1352
			d751 = snap1353
			d752 = snap1354
			d753 = snap1355
			d754 = snap1356
			d755 = snap1357
			d756 = snap1358
			d757 = snap1359
			d758 = snap1360
			d759 = snap1361
			d760 = snap1362
			d761 = snap1363
			d762 = snap1364
			d763 = snap1365
			d764 = snap1366
			d765 = snap1367
			d766 = snap1368
			d767 = snap1369
			d768 = snap1370
			d769 = snap1371
			d770 = snap1372
			d771 = snap1373
			d772 = snap1374
			d773 = snap1375
			d774 = snap1376
			d775 = snap1377
			d776 = snap1378
			d777 = snap1379
			d778 = snap1380
			d779 = snap1381
			d780 = snap1382
			d781 = snap1383
			d782 = snap1384
			d783 = snap1385
			d784 = snap1386
			d785 = snap1387
			d786 = snap1388
			d787 = snap1389
			d1085 = snap1390
			d1086 = snap1391
			d1087 = snap1392
			d1088 = snap1393
			d1089 = snap1394
			d1090 = snap1395
			d1091 = snap1396
			d1092 = snap1397
			d1093 = snap1398
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps1096)
			}
			return result
			ctx.FreeDesc(&d1092)
			return result
			}
			ps1400 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps1400)
			ctx.MarkLabel(lbl0)
			d1401 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d1401)
			ctx.BindReg(r1, &d1401)
			ctx.EmitMovPairToResult(&d1401, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.PatchInt32(r7, int32(96))
			ctx.EmitAddRSP32(int32(96))
			return result
}

func (s *StoragePrefix) prepare() {
	// set up scan
	s.prefixes.prepare()
	s.values.prepare()
}
func (s *StoragePrefix) scan(i uint32, value scm.Scmer) {
	if value.IsNil() {
		s.values.scan(i, scm.NewNil())
		return
	}
	v := scm.String(value)

	for pfid := len(s.prefixdictionary) - 1; pfid >= 0; pfid-- {
		if strings.HasPrefix(v, s.prefixdictionary[pfid]) {
			// learn the string stripped from its prefix
			s.prefixes.scan(i, scm.NewInt(int64(pfid)))
			s.values.scan(i, scm.NewString(v[len(s.prefixdictionary[pfid]):]))
			return
		}
	}
}
func (s *StoragePrefix) init(i uint32) {
	s.prefixes.init(i)
	s.values.init(i)
}
func (s *StoragePrefix) build(i uint32, value scm.Scmer) {
	// store
	if value.IsNil() {
		s.values.build(i, scm.NewNil())
		return
	}
	v := scm.String(value)

	for pfid := len(s.prefixdictionary) - 1; pfid >= 0; pfid-- {
		if strings.HasPrefix(v, s.prefixdictionary[pfid]) {
			// learn the string stripped from its prefix
			s.prefixes.build(i, scm.NewInt(int64(pfid)))
			s.values.build(i, scm.NewString(v[len(s.prefixdictionary[pfid]):]))
			return
		}
	}
}
func (s *StoragePrefix) finish() {
	s.prefixes.finish()
	s.values.finish()
}
func (s *StoragePrefix) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	// TODO: if s.values proposes a StoragePrefix, build it into our cascade??
	return nil
}
