-- Copyright (C) 2026 Carl-Philip Hänsch
-- SPDX-License-Identifier: GPL-3.0-or-later
-- Backup statements have no native implementation; importing one must stop.
BACKUP DATABASE [memcp-tests] TO DISK=N'unused.bak';
GO
INSERT dbo.tsdump_console_lifecycle (id,label) VALUES (998,N'after unsupported');
