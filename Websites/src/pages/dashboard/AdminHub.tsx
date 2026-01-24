import { useEffect, useMemo, useState } from 'react';
import { Shield, Loader2, Save } from 'lucide-react';
import { adminApi, AdminUser } from '@/lib/api';
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

  useEffect(() => {
    fetchUsers();
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
    </div>
  );
}
