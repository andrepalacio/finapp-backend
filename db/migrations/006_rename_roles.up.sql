ALTER TABLE workspace_members DROP CONSTRAINT workspace_members_role_check;
UPDATE workspace_members SET role = 'editor' WHERE role = 'admin';
UPDATE workspace_members SET role = 'viewer' WHERE role = 'member';
ALTER TABLE workspace_members ADD CONSTRAINT workspace_members_role_check CHECK (role IN ('owner', 'editor', 'viewer'));

ALTER TABLE workspace_invitations DROP CONSTRAINT workspace_invitations_role_check;
UPDATE workspace_invitations SET role = 'editor' WHERE role = 'admin';
UPDATE workspace_invitations SET role = 'viewer' WHERE role = 'member';
ALTER TABLE workspace_invitations ALTER COLUMN role SET DEFAULT 'viewer';
ALTER TABLE workspace_invitations ADD CONSTRAINT workspace_invitations_role_check CHECK (role IN ('editor', 'viewer'));
