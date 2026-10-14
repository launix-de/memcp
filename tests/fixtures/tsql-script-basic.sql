-- Copyright (C) 2026 Carl-Philip Hänsch
-- SPDX-License-Identifier: GPL-3.0-or-later
DROP TABLE IF EXISTS dbo.script_import;
CREATE TABLE dbo.script_import(id INT PRIMARY KEY,label NVARCHAR(40));
GO
DECLARE @id INT=7;
INSERT INTO dbo.script_import VALUES(@id,N'line one
GO
line two');
GO
