ALTER TABLE workspace_invitations DROP CONSTRAINT workspace_invitations_role_check;
ALTER TABLE workspace_invitations ALTER COLUMN role SET DEFAULT 'member';
UPDATE workspace_invitations SET role = 'admin' WHERE role = 'editor';
UPDATE workspace_invitations SET role = 'member' WHERE role = 'viewer';
ALTER TABLE workspace_invitations ADD CONSTRAINT workspace_invitations_role_check CHECK (role IN ('admin', 'member'));

ALTER TABLE workspace_members DROP CONSTRAINT workspace_members_role_check;
UPDATE workspace_members SET role = 'admin' WHERE role = 'editor';
UPDATE workspace_members SET role = 'member' WHERE role = 'viewer';
ALTER TABLE workspace_members ADD CONSTRAINT workspace_members_role_check CHECK (role IN ('owner', 'admin', 'member'));
