-- Copyright (C) 2026 Carl-Philip Hänsch
-- SPDX-License-Identifier: GPL-3.0-or-later
BEGIN TRANSACTION;
INSERT dbo.tsdump_console_lifecycle (id,label) VALUES (999,N'uncommitted');
