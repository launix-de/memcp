-- Copyright (C) 2026 Carl-Philip Hänsch
-- SPDX-License-Identifier: GPL-3.0-or-later
CREATE PROCEDURE tsdump_procedure AS SELECT 1;
GO
INSERT dbo.tsdump_items (tenant,id,name) VALUES (1,998,N'after unsupported');
