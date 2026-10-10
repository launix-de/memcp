-- Copyright (C) 2026 Carl-Philip Hänsch
-- SPDX-License-Identifier: GPL-3.0-or-later
USE [export_source]
GO
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE TABLE [dbo].[tsdump_items] (
 [tenant] SMALLINT NOT NULL,
 [id] INT IDENTITY(1,1) NOT NULL,
 [name] NVARCHAR(80) NOT NULL,
 [note] NVARCHAR(MAX) NULL,
 [enabled] SMALLINT NOT NULL DEFAULT (-1),
 CONSTRAINT [PK_items] PRIMARY KEY CLUSTERED ([tenant] ASC, [id] DESC)
)
GO
SET IDENTITY_INSERT [dbo].[tsdump_items] ON
INSERT [export_source].[dbo].[tsdump_items] ([tenant],[id],[name],[note]) VALUES (1,7,N'Grüße 🐘',N'export_source.dbo.keep')
INSERT [dbo].[tsdump_items] ([tenant],[id],[name],[note]) VALUES (1,15,N'Semi;colon ''quote''',N'first
GO
last')
GO -- separator outside a value
SET IDENTITY_INSERT [dbo].[tsdump_items] OFF
GO
CREATE TABLE dbo.tsdump_identity (id INT IDENTITY(1,1) PRIMARY KEY, value INT)
CREATE TABLE dbo.tsdump_identity_observation (stage INT PRIMARY KEY, value NUMERIC(38,0))
INSERT dbo.tsdump_identity (value) VALUES (10),(20)
INSERT dbo.tsdump_identity_observation VALUES (1,SCOPE_IDENTITY())
INSERT dbo.tsdump_identity (value) VALUES (30)
GO
INSERT dbo.tsdump_identity_observation VALUES (2,SCOPE_IDENTITY()),(3,@@IDENTITY)
GO
-- Conditional declarations retain their leaf across line boundaries.
IF OBJECT_ID(N'dbo.tsdump_conditional',N'U') IS NOT NULL
 DROP TABLE dbo.tsdump_conditional
IF OBJECT_ID(N'dbo.tsdump_conditional',N'U') IS NULL
 CREATE TABLE export_source.dbo.tsdump_conditional(id INT PRIMARY KEY, label NVARCHAR(80))
GO
IF 1=0
 INSERT dbo.tsdump_conditional VALUES(1,N'wrong')
ELSE
 INSERT dbo.tsdump_conditional VALUES(2,N'ELSE; CREATE 🐘')
GO
