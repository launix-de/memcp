-- Copyright (C) 2026 Carl-Philip Hänsch
-- SPDX-License-Identifier: GPL-3.0-or-later
BEGIN TRANSACTION;
INSERT dbo.tsdump_items (tenant,id,name) VALUES (1,999,N'uncommitted');
