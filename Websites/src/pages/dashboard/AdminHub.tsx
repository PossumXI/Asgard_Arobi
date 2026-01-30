import { useEffect, useMemo, useState } from 'react';
import { Shield, Loader2, Save, Key, RefreshCw, UserPlus, Trash2 } from 'lucide-react';
import { adminApi, AdminUser, AccessCodeRecord } from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';

type DraftState = Record<string, Partial<Pick<AdminUser, 'fullName' | 'subscriptionTier' | 'isGovernment'>>>;

const tierOptions: AdminUser['subscriptionTier'][] = ['observer', 'supporter', 'commander'];

export default function AdminHub() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [drafts, setDrafts] = useState<DraftState>({});
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [accessCodes, setAccessCodes] = useState<AccessCodeRecord[]>([]);
  const [isCodesLoading, setIsCodesLoading] = useState(false);
  const [codeError, setCodeError] = useState<string | null>(null);
  const [isIssuingCode, setIsIssuingCode] = useState(false);
  const [issueEmail, setIssueEmail] = useState('');
  const [issueClearance, setIssueClearance] = useState('government');
  const [issueScope, setIssueScope] = useState('all');
  const [isRotatingAll, setIsRotatingAll] = useState(false);
  const [isRotatingUser, setIsRotatingUser] = useState<string | null>(null);
  const [isRevoking, setIsRevoking] = useState<string | null>(null);
  const [createUserData, setCreateUserData] = useState({
    email: 'Gaetano@aura-genesis.org',
    fullName: 'Gaetano Comparcola',
    subscriptionTier: 'commander' as AdminUser['subscriptionTier'],
    isGovernment: true,
    createAccessCode: true,
    clearanceLevel: 'government',
  });
  const [createUserResult, setCreateUserResult] = useState<{ password: string; accessCode?: string } | null>(null);

  const fetchUsers = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await adminApi.listUsers();
      setUsers(data);
    } catch (err) {
      setError('Failed to load users');
    } finally {
      setIsLoading(false);
    }
  };

  const fetchAccessCodes = async () => {
    setIsCodesLoading(true);
    setCodeError(null);
    try {
      const data = await adminApi.listAccessCodes();
      setAccessCodes(data);
    } catch (err) {
      setCodeError('Failed to load access codes');
    } finally {
      setIsCodesLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
    fetchAccessCodes();
  }, []);

  const handleDraftChange = (userId: string, field: keyof AdminUser, value: string | boolean) => {
    setDrafts((prev) => ({
      ...prev,
      [userId]: {
        ...prev[userId],
        [field]: value,
      },
    }));
  };

  const handleSave = async (userId: string) => {
    const draft = drafts[userId];
    if (!draft) return;
    setIsSaving(userId);
    try {
      const updated = await adminApi.updateUser(userId, draft);
      setUsers((prev) => prev.map((u) => (u.id === userId ? updated : u)));
      setDrafts((prev) => {
        const next = { ...prev };
        delete next[userId];
        return next;
      });
    } catch (err) {
      setError('Failed to update user');
    } finally {
      setIsSaving(null);
    }
  };

  const handleCreateUser = async () => {
    setError(null);
    setCreateUserResult(null);
    try {
      const result = await adminApi.createUser(createUserData);
      setCreateUserResult({
        password: result.temporaryPassword,
        accessCode: result.accessCode,
      });
      await fetchUsers();
      await fetchAccessCodes();
    } catch (err) {
      setError('Failed to create user');
    }
  };

  const handleIssueCode = async () => {
    setCodeError(null);
    setIsIssuingCode(true);
    try {
      await adminApi.issueAccessCode({
        email: issueEmail,
        clearanceLevel: issueClearance,
        scope: issueScope,
      });
      setIssueEmail('');
      await fetchAccessCodes();
    } catch (err) {
      setCodeError('Failed to issue access code');
    } finally {
      setIsIssuingCode(false);
    }
  };

  const handleRotateAll = async () => {
    setCodeError(null);
    setIsRotatingAll(true);
    try {
      await adminApi.rotateAllAccessCodes();
      await fetchAccessCodes();
    } catch (err) {
      setCodeError('Failed to rotate access codes');
    } finally {
      setIsRotatingAll(false);
    }
  };

  const handleRotateUser = async (email?: string | null) => {
    if (!email) return;
    setCodeError(null);
    setIsRotatingUser(email);
    try {
      await adminApi.rotateAccessCode({ email });
      await fetchAccessCodes();
    } catch (err) {
      setCodeError('Failed to rotate access code');
    } finally {
      setIsRotatingUser(null);
    }
  };

  const handleRevoke = async (codeId: string) => {
    setCodeError(null);
    setIsRevoking(codeId);
    try {
      await adminApi.revokeAccessCode(codeId);
      await fetchAccessCodes();
    } catch (err) {
      setCodeError('Failed to revoke access code');
    } finally {
      setIsRevoking(null);
    }
  };

  const hasChanges = useMemo(() => new Set(Object.keys(drafts)), [drafts]);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-row items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
            <Shield className="w-5 h-5 text-primary" />
          </div>
          <div>
            <CardTitle>Admin Access Control</CardTitle>
            <p className="text-sm text-asgard-500">
              Manage tier access and government flags for users.
            </p>
          </div>
        </CardHeader>
      </Card>

      {isLoading ? (
        <div className="flex items-center justify-center py-10">
          <Loader2 className="w-6 h-6 animate-spin text-primary" />
        </div>
      ) : error ? (
        <div className="text-sm text-red-500">{error}</div>
      ) : (
        <div className="space-y-4">
          {users.map((user) => {
            const draft = drafts[user.id] ?? {};
            const subscriptionTier = (draft.subscriptionTier ?? user.subscriptionTier) as AdminUser['subscriptionTier'];
            const isGovernment = draft.isGovernment ?? user.isGovernment;
            const fullName = draft.fullName ?? user.fullName;
            const dirty = hasChanges.has(user.id);

            return (
              <Card key={user.id}>
                <CardContent className="p-6 space-y-4">
                  <div className="flex flex-wrap items-center justify-between gap-4">
                    <div>
                      <div className="text-sm text-asgard-500">{user.email}</div>
                      <div className="text-lg font-semibold text-asgard-900 dark:text-white">
                        {fullName || 'Unnamed user'}
                      </div>
                    </div>
                    <Button
                      onClick={() => handleSave(user.id)}
                      disabled={!dirty || isSaving === user.id}
                      className="flex items-center gap-2"
                    >
                      {isSaving === user.id ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                      Save
                    </Button>
                  </div>

                  <div className="grid gap-4 md:grid-cols-3">
                    <div>
                      <label className="text-xs font-semibold uppercase tracking-wide text-asgard-500">
                        Full Name
                      </label>
                      <Input
                        value={fullName ?? ''}
                        onChange={(e) => handleDraftChange(user.id, 'fullName', e.target.value)}
                      />
                    </div>
                    <div>
                      <label className="text-xs font-semibold uppercase tracking-wide text-asgard-500">
                        Subscription Tier
                      </label>
                      <select
                        value={subscriptionTier}
                        onChange={(e) => handleDraftChange(user.id, 'subscriptionTier', e.target.value)}
                        className="mt-1 w-full h-10 rounded-xl border border-asgard-200 dark:border-asgard-700 bg-white dark:bg-asgard-900 px-3 text-sm"
                      >
                        {tierOptions.map((tier) => (
                          <option key={tier} value={tier}>
                            {tier}
                          </option>
                        ))}
                      </select>
                    </div>
                    <div className="flex items-end gap-3">
                      <label className="flex items-center gap-2 text-sm text-asgard-600 dark:text-asgard-300">
                        <input
                          type="checkbox"
                          checked={!!isGovernment}
                          onChange={(e) => handleDraftChange(user.id, 'isGovernment', e.target.checked)}
                          className="h-4 w-4 rounded border border-asgard-300 dark:border-asgard-600"
                        />
                        Government Access
                      </label>
                    </div>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}

      <Card>
        <CardHeader className="flex flex-row items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
            <UserPlus className="w-5 h-5 text-primary" />
          </div>
          <div>
            <CardTitle>Create Profile</CardTitle>
            <p className="text-sm text-asgard-500">
              Create a new user profile and optional clearance code.
            </p>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <Input
              label="Email"
              value={createUserData.email}
              onChange={(e) => setCreateUserData((prev) => ({ ...prev, email: e.target.value }))}
            />
            <Input
              label="Full Name"
              value={createUserData.fullName}
              onChange={(e) => setCreateUserData((prev) => ({ ...prev, fullName: e.target.value }))}
            />
            <div>
              <label className="text-xs font-semibold uppercase tracking-wide text-asgard-500">
                Subscription Tier
              </label>
              <select
                value={createUserData.subscriptionTier}
                onChange={(e) =>
                  setCreateUserData((prev) => ({
                    ...prev,
                    subscriptionTier: e.target.value as AdminUser['subscriptionTier'],
                  }))
                }
                className="mt-1 w-full h-10 rounded-xl border border-asgard-200 dark:border-asgard-700 bg-white dark:bg-asgard-900 px-3 text-sm"
              >
                {tierOptions.map((tier) => (
                  <option key={tier} value={tier}>
                    {tier}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex items-end gap-3">
              <label className="flex items-center gap-2 text-sm text-asgard-600 dark:text-asgard-300">
                <input
                  type="checkbox"
                  checked={createUserData.isGovernment}
                  onChange={(e) => setCreateUserData((prev) => ({ ...prev, isGovernment: e.target.checked }))}
                  className="h-4 w-4 rounded border border-asgard-300 dark:border-asgard-600"
                />
                Government Access
              </label>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-4">
            <label className="flex items-center gap-2 text-sm text-asgard-600 dark:text-asgard-300">
              <input
                type="checkbox"
                checked={createUserData.createAccessCode}
                onChange={(e) => setCreateUserData((prev) => ({ ...prev, createAccessCode: e.target.checked }))}
                className="h-4 w-4 rounded border border-asgard-300 dark:border-asgard-600"
              />
              Issue access code
            </label>
            <Input
              label="Clearance"
              value={createUserData.clearanceLevel}
              onChange={(e) => setCreateUserData((prev) => ({ ...prev, clearanceLevel: e.target.value }))}
            />
            <Button onClick={handleCreateUser} className="flex items-center gap-2">
              <UserPlus className="w-4 h-4" />
              Create Profile
            </Button>
          </div>
          {createUserResult && (
            <div className="rounded-xl border border-asgard-200 dark:border-asgard-700 p-4 text-sm">
              <p className="text-asgard-700 dark:text-asgard-300">
                Temporary password: <span className="font-semibold">{createUserResult.password}</span>
              </p>
              {createUserResult.accessCode && (
                <p className="text-asgard-700 dark:text-asgard-300">
                  Access code: <span className="font-semibold">{createUserResult.accessCode}</span>
                </p>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
            <Key className="w-5 h-5 text-primary" />
          </div>
          <div>
            <CardTitle>Access Code Management</CardTitle>
            <p className="text-sm text-asgard-500">
              Issue, rotate, and revoke clearance codes.
            </p>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-3">
            <Input
              label="User Email"
              value={issueEmail}
              onChange={(e) => setIssueEmail(e.target.value)}
              placeholder="user@aura-genesis.org"
            />
            <Input
              label="Clearance Level"
              value={issueClearance}
              onChange={(e) => setIssueClearance(e.target.value)}
            />
            <Input
              label="Scope"
              value={issueScope}
              onChange={(e) => setIssueScope(e.target.value)}
            />
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <Button
              onClick={handleIssueCode}
              disabled={isIssuingCode || issueEmail.trim() === ''}
              className="flex items-center gap-2"
            >
              {isIssuingCode ? <Loader2 className="w-4 h-4 animate-spin" /> : <Key className="w-4 h-4" />}
              Issue Code
            </Button>
            <Button
              variant="outline"
              onClick={handleRotateAll}
              disabled={isRotatingAll}
              className="flex items-center gap-2"
            >
              {isRotatingAll ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
              Rotate All
            </Button>
            <Button
              variant="outline"
              onClick={fetchAccessCodes}
              className="flex items-center gap-2"
            >
              <RefreshCw className="w-4 h-4" />
              Refresh
            </Button>
          </div>
          {codeError && <div className="text-sm text-red-500">{codeError}</div>}
          {isCodesLoading ? (
            <div className="flex items-center justify-center py-6">
              <Loader2 className="w-5 h-5 animate-spin text-primary" />
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="text-left text-asgard-500">
                  <tr>
                    <th className="py-2">User</th>
                    <th className="py-2">Clearance</th>
                    <th className="py-2">Scope</th>
                    <th className="py-2">Status</th>
                    <th className="py-2">Expires</th>
                    <th className="py-2">Last Used</th>
                    <th className="py-2">Code</th>
                    <th className="py-2">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {accessCodes.map((code) => {
                    const status = code.revokedAt
                      ? 'Revoked'
                      : new Date(code.expiresAt) < new Date()
                      ? 'Expired'
                      : 'Active';
                    return (
                      <tr key={code.id} className="border-t border-asgard-100 dark:border-asgard-800">
                        <td className="py-2 pr-4">
                          <div className="text-asgard-900 dark:text-white">{code.userEmail ?? 'Unassigned'}</div>
                          <div className="text-xs text-asgard-500">{code.userFullName ?? ''}</div>
                        </td>
                        <td className="py-2">{code.clearanceLevel}</td>
                        <td className="py-2">{code.scope}</td>
                        <td className="py-2">{status}</td>
                        <td className="py-2">{new Date(code.expiresAt).toLocaleString()}</td>
                        <td className="py-2">{code.lastUsedAt ? new Date(code.lastUsedAt).toLocaleString() : '-'}</td>
                        <td className="py-2">****{code.codeLast4}</td>
                        <td className="py-2">
                          <div className="flex items-center gap-2">
                            <Button
                              variant="outline"
                              onClick={() => handleRotateUser(code.userEmail)}
                              disabled={isRotatingUser === code.userEmail}
                              className="h-8 px-2"
                            >
                              {isRotatingUser === code.userEmail ? (
                                <Loader2 className="w-4 h-4 animate-spin" />
                              ) : (
                                <RefreshCw className="w-4 h-4" />
                              )}
                            </Button>
                            <Button
                              variant="outline"
                              onClick={() => handleRevoke(code.id)}
                              disabled={isRevoking === code.id}
                              className="h-8 px-2"
                            >
                              {isRevoking === code.id ? (
                                <Loader2 className="w-4 h-4 animate-spin" />
                              ) : (
                                <Trash2 className="w-4 h-4" />
                              )}
                            </Button>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
