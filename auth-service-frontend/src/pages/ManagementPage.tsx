import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useUserSync, hasPermission } from '../hooks/useUserSync'
import type { UserRole } from '../hooks/useUserSync'
import Navbar from '../components/Navbar'
import '../App.css'
import './ManagementPage.css'

const API_SERVER_URL = import.meta.env.VITE_API_SERVER_URL || 'http://localhost:8080'

interface User {
  id: string
  email: string
  name: string
  role: UserRole
  status: string
  created_at: string
}

interface APIKey {
  id: number
  key_id: string
  scopes: string[]
  revoked: boolean
  expires_at: string | null
  created_at: string
}

export default function ManagementPage() {
  const navigate = useNavigate()
  const { role, userId, tenantId, pricingPlan, isSynced } = useUserSync()

  const [users, setUsers] = useState<User[]>([])
  const [apiKeys, setApiKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [successMessage, setSuccessMessage] = useState<string | null>(null)

  // Modal states
  const [showInviteModal, setShowInviteModal] = useState(false)
  const [showTransferModal, setShowTransferModal] = useState(false)
  const [showDeleteTenantModal, setShowDeleteTenantModal] = useState(false)
  const [showNewAPIKeyModal, setShowNewAPIKeyModal] = useState(false)
  const [newAPIKey, setNewAPIKey] = useState<string | null>(null)

  // Form states
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteName, setInviteName] = useState('')
  const [inviteRole, setInviteRole] = useState<'viewer' | 'editor'>('viewer')
  const [transferUserId, setTransferUserId] = useState('')
  const [deleteConfirmText, setDeleteConfirmText] = useState('')
  const [invitationURL, setInvitationURL] = useState<string | null>(null)
  const [invitationCopied, setInvitationCopied] = useState(false)

  const isOwner = role === 'owner'
  const isAdmin = hasPermission(role, 'admin')

  // Fetch headers with user authentication
  const getHeaders = () => ({
    'Content-Type': 'application/json',
    'X-User-ID': userId || '',
  })

  // Fetch users
  const fetchUsers = async () => {
    try {
      const response = await fetch(`${API_SERVER_URL}/v1/users`, {
        headers: getHeaders(),
      })
      if (response.ok) {
        const data = await response.json()
        console.log('API Response:', JSON.stringify(data, null, 2))

        // Handle different possible response structures
        let usersList: User[] = []
        if (Array.isArray(data)) {
          usersList = data
        } else if (data.users && Array.isArray(data.users)) {
          usersList = data.users
        }

        console.log('Users to render:', usersList.length, 'users')
        usersList.forEach((u, i) => {
          console.log(`User ${i}: name="${u.name}", email="${u.email}", role="${u.role}"`)
        })
        setUsers(usersList)
      } else {
        const errorData = await response.json().catch(() => ({}))
        console.error('Failed to fetch users:', response.status, errorData)
      }
    } catch (err) {
      console.error('Failed to fetch users:', err)
    }
  }

  // Fetch API keys
  const fetchAPIKeys = async () => {
    try {
      const response = await fetch(`${API_SERVER_URL}/v1/admin/api_keys`, {
        headers: getHeaders(),
      })
      if (response.ok) {
        const data = await response.json()
        setApiKeys(data.api_keys || [])
      }
    } catch (err) {
      console.error('Failed to fetch API keys:', err)
    }
  }

  useEffect(() => {
    if (!isSynced) return

    if (!isAdmin) {
      navigate('/dashboard')
      return
    }

    const loadData = async () => {
      setLoading(true)
      await Promise.all([fetchUsers(), fetchAPIKeys()])
      setLoading(false)
    }
    loadData()
  }, [isSynced, isAdmin, navigate])

  const showSuccess = (message: string) => {
    setSuccessMessage(message)
    setTimeout(() => setSuccessMessage(null), 3000)
  }

  const showError = (message: string) => {
    setError(message)
    setTimeout(() => setError(null), 5000)
  }

  // User management actions
  const handleInviteUser = async () => {
    try {
      const response = await fetch(`${API_SERVER_URL}/v1/admin/users/invite`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          email: inviteEmail,
          name: inviteName,
          role: inviteRole,
        }),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || data.error || 'Failed to invite user')
      }

      const data = await response.json()
      setInvitationURL(data.invitation_url || null)
      showSuccess(`Invitation sent to ${inviteEmail}`)
      // Don't close modal yet - show invitation URL
      fetchUsers()
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to invite user')
    }
  }

  const handleSuspendUser = async (targetUserId: string) => {
    if (!confirm('Are you sure you want to suspend this user?')) return

    try {
      const response = await fetch(`${API_SERVER_URL}/v1/admin/users/${targetUserId}/suspend`, {
        method: 'PATCH',
        headers: getHeaders(),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || data.error || 'Failed to suspend user')
      }

      showSuccess('User suspended')
      fetchUsers()
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to suspend user')
    }
  }

  const handleUnsuspendUser = async (targetUserId: string) => {
    try {
      const response = await fetch(`${API_SERVER_URL}/v1/admin/users/${targetUserId}/unsuspend`, {
        method: 'PATCH',
        headers: getHeaders(),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || data.error || 'Failed to unsuspend user')
      }

      showSuccess('User unsuspended')
      fetchUsers()
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to unsuspend user')
    }
  }

  const handleRemoveUser = async (targetUserId: string) => {
    if (!confirm('Are you sure you want to remove this user? This cannot be undone.')) return

    try {
      const response = await fetch(`${API_SERVER_URL}/v1/admin/users/${targetUserId}`, {
        method: 'DELETE',
        headers: getHeaders(),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || data.error || 'Failed to remove user')
      }

      showSuccess('User removed')
      fetchUsers()
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to remove user')
    }
  }

  const handleUpdateRole = async (targetUserId: string, newRole: 'viewer' | 'editor') => {
    try {
      const response = await fetch(`${API_SERVER_URL}/v1/admin/users/${targetUserId}/role`, {
        method: 'PATCH',
        headers: getHeaders(),
        body: JSON.stringify({ role: newRole }),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || data.error || 'Failed to update role')
      }

      showSuccess(`User role updated to ${newRole}`)
      fetchUsers()
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to update role')
    }
  }

  const handlePromoteToAdmin = async (targetUserId: string) => {
    if (!confirm('Are you sure you want to promote this user to admin?')) return

    try {
      const response = await fetch(`${API_SERVER_URL}/v1/owner/users/${targetUserId}/promote-admin`, {
        method: 'POST',
        headers: getHeaders(),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || data.error || 'Failed to promote to admin')
      }

      showSuccess('User promoted to admin')
      fetchUsers()
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to promote to admin')
    }
  }

  const handleDemoteAdmin = async (targetUserId: string) => {
    if (!confirm('Are you sure you want to demote this admin to editor?')) return

    try {
      const response = await fetch(`${API_SERVER_URL}/v1/owner/users/${targetUserId}/demote-admin`, {
        method: 'DELETE',
        headers: getHeaders(),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || data.error || 'Failed to demote admin')
      }

      showSuccess('Admin demoted to editor')
      fetchUsers()
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to demote admin')
    }
  }

  // API Key actions
  const handleCreateAPIKey = async () => {
    try {
      const expiresAt = new Date()
      expiresAt.setFullYear(expiresAt.getFullYear() + 1)

      const response = await fetch(`${API_SERVER_URL}/v1/admin/api_keys`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          tenant_id: tenantId,
          scopes: ['*'],
          expires_at: expiresAt.toISOString(),
        }),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || data.error || 'Failed to create API key')
      }

      const data = await response.json()
      setNewAPIKey(`${data.key_id}:${data.secret}`)
      setShowNewAPIKeyModal(true)
      fetchAPIKeys()
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to create API key')
    }
  }

  const handleRevokeAPIKey = async (keyId: string) => {
    if (!confirm('Are you sure you want to revoke this API key? This cannot be undone.')) return

    try {
      const response = await fetch(`${API_SERVER_URL}/v1/admin/api_keys/${keyId}`, {
        method: 'DELETE',
        headers: getHeaders(),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || data.error || 'Failed to revoke API key')
      }

      showSuccess('API key revoked')
      fetchAPIKeys()
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to revoke API key')
    }
  }

  // Owner-only actions
  const handleTransferOwnership = async () => {
    if (!transferUserId) {
      showError('Please select a user to transfer ownership to')
      return
    }

    try {
      const response = await fetch(`${API_SERVER_URL}/v1/owner/transfer-ownership`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ new_owner_id: transferUserId }),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || data.error || 'Failed to transfer ownership')
      }

      showSuccess('Ownership transferred. You are now an admin.')
      setShowTransferModal(false)
      setTransferUserId('')
      fetchUsers()
      // Refresh the page to update the role
      window.location.reload()
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to transfer ownership')
    }
  }

  const handleDeleteTenant = async () => {
    if (deleteConfirmText !== 'DELETE') {
      showError('Please type DELETE to confirm')
      return
    }

    try {
      const response = await fetch(`${API_SERVER_URL}/v1/owner/tenants/${tenantId}`, {
        method: 'DELETE',
        headers: getHeaders(),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || data.error || 'Failed to delete tenant')
      }

      // Clear localStorage and redirect to home
      localStorage.clear()
      window.location.href = '/'
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to delete tenant')
    }
  }

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setInvitationCopied(true)
      setTimeout(() => setInvitationCopied(false), 2000)
    } catch {
      showError('Failed to copy')
    }
  }

  const handleCloseInviteModal = () => {
    setShowInviteModal(false)
    setInviteEmail('')
    setInviteName('')
    setInviteRole('viewer')
    setInvitationURL(null)
    setInvitationCopied(false)
  }

  if (!isSynced || loading) {
    return (
      <div className="page-container">
        <Navbar />
        <div className="page-content">
          <div className="management-loading">Loading...</div>
        </div>
      </div>
    )
  }

  return (
    <div className="page-container">
      <Navbar />
      <div className="page-content">
        <div className="management-container">
          <div className="management-header">
            <h1>Management</h1>
            <div className="management-role-badge">
              <span className={`role-badge role-badge-${role}`}>
                {role?.charAt(0).toUpperCase()}{role?.slice(1)}
              </span>
            </div>
          </div>

          {/* Success/Error Messages */}
          {successMessage && (
            <div className="management-message management-success">{successMessage}</div>
          )}
          {error && (
            <div className="management-message management-error">{error}</div>
          )}

          {/* Team Members Section */}
          <section className="management-section">
            <div className="section-header">
              <h2>Team Members</h2>
              <button className="btn btn-primary" onClick={() => setShowInviteModal(true)}>
                Invite User
              </button>
            </div>
            <div className="users-table-container">
              <table className="users-table">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Email</th>
                    <th>Role</th>
                    <th>Status</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="empty-state">
                        No users found.
                      </td>
                    </tr>
                  ) : (
                    users.map((user) => {
                      console.log('Rendering row for:', user.id, user.name, user.email)
                      return (
                      <tr key={user.id} className={user.id === userId ? 'current-user' : ''}>
                        <td>{user.name || user.email?.split('@')[0] || 'Unknown'}</td>
                        <td>{user.email || 'No email'}</td>
                      <td>
                        <span className={`role-badge role-badge-${user.role}`}>
                          {user.role}
                        </span>
                      </td>
                      <td>
                        <span className={`status-badge status-badge-${user.status}`}>
                          {user.status}
                        </span>
                      </td>
                      <td className="actions-cell">
                        {user.id !== userId && (
                          <>
                            {/* Admin can manage viewers and editors */}
                            {user.role === 'viewer' && (
                              <button
                                className="btn btn-small btn-secondary"
                                onClick={() => handleUpdateRole(user.id, 'editor')}
                                title="Promote to Editor"
                              >
                                Promote
                              </button>
                            )}
                            {user.role === 'editor' && (
                              <>
                                <button
                                  className="btn btn-small btn-secondary"
                                  onClick={() => handleUpdateRole(user.id, 'viewer')}
                                  title="Demote to Viewer"
                                >
                                  Demote
                                </button>
                                {isOwner && (
                                  <button
                                    className="btn btn-small btn-primary"
                                    onClick={() => handlePromoteToAdmin(user.id)}
                                    title="Promote to Admin"
                                  >
                                    Make Admin
                                  </button>
                                )}
                              </>
                            )}
                            {user.role === 'admin' && isOwner && (
                              <button
                                className="btn btn-small btn-warning"
                                onClick={() => handleDemoteAdmin(user.id)}
                                title="Demote to Editor"
                              >
                                Demote
                              </button>
                            )}
                            {/* Suspend/Unsuspend */}
                            {user.role !== 'owner' && (user.role !== 'admin' || isOwner) && (
                              user.status === 'active' ? (
                                <button
                                  className="btn btn-small btn-warning"
                                  onClick={() => handleSuspendUser(user.id)}
                                  title="Suspend User"
                                >
                                  Suspend
                                </button>
                              ) : user.status === 'suspended' && (
                                <button
                                  className="btn btn-small btn-secondary"
                                  onClick={() => handleUnsuspendUser(user.id)}
                                  title="Unsuspend User"
                                >
                                  Unsuspend
                                </button>
                              )
                            )}
                            {/* Remove */}
                            {user.role !== 'owner' && (user.role !== 'admin' || isOwner) && (
                              <button
                                className="btn btn-small btn-danger"
                                onClick={() => handleRemoveUser(user.id)}
                                title="Remove User"
                              >
                                Remove
                              </button>
                            )}
                          </>
                        )}
                        {user.id === userId && <span className="you-badge">You</span>}
                      </td>
                    </tr>
                      )
                    })
                  )}
                </tbody>
              </table>
            </div>
          </section>

          {/* API Keys Section */}
          <section className="management-section">
            <div className="section-header">
              <h2>API Keys</h2>
              <button className="btn btn-primary" onClick={handleCreateAPIKey}>
                Create API Key
              </button>
            </div>
            <div className="api-keys-table-container">
              <table className="api-keys-table">
                <thead>
                  <tr>
                    <th>Key ID</th>
                    <th>Created</th>
                    <th>Expires</th>
                    <th>Status</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {apiKeys.map((key) => (
                    <tr key={key.id} className={key.revoked ? 'revoked' : ''}>
                      <td>
                        <code>{key.key_id}</code>
                      </td>
                      <td>{new Date(key.created_at).toLocaleDateString()}</td>
                      <td>
                        {key.expires_at
                          ? new Date(key.expires_at).toLocaleDateString()
                          : 'Never'}
                      </td>
                      <td>
                        <span className={`status-badge ${key.revoked ? 'status-badge-suspended' : 'status-badge-active'}`}>
                          {key.revoked ? 'Revoked' : 'Active'}
                        </span>
                      </td>
                      <td>
                        {!key.revoked && (
                          <button
                            className="btn btn-small btn-danger"
                            onClick={() => handleRevokeAPIKey(key.key_id)}
                          >
                            Revoke
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                  {apiKeys.length === 0 && (
                    <tr>
                      <td colSpan={5} className="empty-state">
                        No API keys found. Create one to get started.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </section>

          {/* Owner-Only Section */}
          {isOwner && (
            <>
              {/* Billing Section */}
              <section className="management-section">
                <div className="section-header">
                  <h2>Billing</h2>
                </div>
                <div className="billing-info">
                  <p>
                    <strong>Current Plan:</strong>{' '}
                    <span className="plan-badge">{pricingPlan || 'Starter'}</span>
                  </p>
                  <button
                    className="btn btn-primary"
                    onClick={() => navigate('/pricing')}
                  >
                    Change Plan
                  </button>
                </div>
              </section>

              {/* Danger Zone */}
              <section className="management-section danger-zone">
                <div className="section-header">
                  <h2>Danger Zone</h2>
                </div>
                <div className="danger-actions">
                  <div className="danger-action">
                    <div>
                      <h3>Transfer Ownership</h3>
                      <p>Transfer ownership of this organization to another admin or user.</p>
                    </div>
                    <button
                      className="btn btn-warning"
                      onClick={() => setShowTransferModal(true)}
                    >
                      Transfer Ownership
                    </button>
                  </div>
                  <div className="danger-action">
                    <div>
                      <h3>Delete Organization</h3>
                      <p>Permanently delete this organization and all its data. This cannot be undone.</p>
                    </div>
                    <button
                      className="btn btn-danger"
                      onClick={() => setShowDeleteTenantModal(true)}
                    >
                      Delete Organization
                    </button>
                  </div>
                </div>
              </section>
            </>
          )}
        </div>
      </div>

      {/* Invite User Modal */}
      {showInviteModal && (
        <div className="modal-overlay" onClick={handleCloseInviteModal}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>Invite User</h2>
            </div>
            <div className="modal-body">
              {!invitationURL ? (
                <>
                  <div className="form-group">
                    <label>Email</label>
                    <input
                      type="email"
                      value={inviteEmail}
                      onChange={(e) => setInviteEmail(e.target.value)}
                      placeholder="user@example.com"
                    />
                  </div>
                  <div className="form-group">
                    <label>Name (optional)</label>
                    <input
                      type="text"
                      value={inviteName}
                      onChange={(e) => setInviteName(e.target.value)}
                      placeholder="John Doe"
                    />
                  </div>
                  <div className="form-group">
                    <label>Role</label>
                    <select
                      value={inviteRole}
                      onChange={(e) => setInviteRole(e.target.value as 'viewer' | 'editor')}
                    >
                      <option value="viewer">Viewer</option>
                      <option value="editor">Editor</option>
                    </select>
                  </div>
                </>
              ) : (
                <>
                  <div className="invitation-success-message">
                    <p>
                      <strong>Invitation sent!</strong> The invited user will receive an invitation link via email.
                      {!invitationURL.includes('localhost') && (
                        <> If they don't receive it, you can copy and share the link below.</>
                      )}
                    </p>
                  </div>
                  {invitationURL && (
                    <div className="invitation-url-section">
                      <label>Invitation URL</label>
                      <div className="invitation-url-display">
                        <input
                          type="text"
                          readOnly
                          value={invitationURL}
                          className="invitation-url-input"
                        />
                        <button
                          className="btn btn-small btn-secondary"
                          onClick={() => copyToClipboard(invitationURL)}
                        >
                          {invitationCopied ? 'Copied!' : 'Copy'}
                        </button>
                      </div>
                    </div>
                  )}
                </>
              )}
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={handleCloseInviteModal}>
                {invitationURL ? 'Close' : 'Cancel'}
              </button>
              {!invitationURL && (
                <button className="btn btn-primary" onClick={handleInviteUser}>
                  Send Invitation
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* New API Key Modal */}
      {showNewAPIKeyModal && newAPIKey && (
        <div className="modal-overlay">
          <div className="modal-content">
            <div className="modal-header">
              <h2>API Key Created</h2>
            </div>
            <div className="modal-body">
              <div className="warning-box">
                <span className="warning-icon">!</span>
                <p>
                  <strong>Important:</strong> This is the only time you will see this API key.
                  Please copy and save it securely.
                </p>
              </div>
              <div className="api-key-display">
                <code>{newAPIKey}</code>
                <button className="btn btn-small btn-secondary" onClick={() => copyToClipboard(newAPIKey)}>
                  Copy
                </button>
              </div>
            </div>
            <div className="modal-footer">
              <button
                className="btn btn-primary"
                onClick={() => {
                  setShowNewAPIKeyModal(false)
                  setNewAPIKey(null)
                }}
              >
                I've saved my API key
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Transfer Ownership Modal */}
      {showTransferModal && (
        <div className="modal-overlay" onClick={() => setShowTransferModal(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>Transfer Ownership</h2>
            </div>
            <div className="modal-body">
              <div className="warning-box">
                <span className="warning-icon">!</span>
                <p>
                  <strong>Warning:</strong> You will be demoted to admin after transferring ownership.
                </p>
              </div>
              <div className="form-group">
                <label>Select new owner</label>
                <select
                  value={transferUserId}
                  onChange={(e) => setTransferUserId(e.target.value)}
                >
                  <option value="">Select a user...</option>
                  {users
                    .filter((u) => u.id !== userId && u.status === 'active')
                    .map((u) => (
                      <option key={u.id} value={u.id}>
                        {u.name || u.email} ({u.role})
                      </option>
                    ))}
                </select>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowTransferModal(false)}>
                Cancel
              </button>
              <button className="btn btn-warning" onClick={handleTransferOwnership}>
                Transfer Ownership
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Tenant Modal */}
      {showDeleteTenantModal && (
        <div className="modal-overlay" onClick={() => setShowDeleteTenantModal(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>Delete Organization</h2>
            </div>
            <div className="modal-body">
              <div className="warning-box warning-box-danger">
                <span className="warning-icon">!</span>
                <p>
                  <strong>This action cannot be undone.</strong> All users, API keys, and data
                  will be permanently deleted.
                </p>
              </div>
              <div className="form-group">
                <label>Type DELETE to confirm</label>
                <input
                  type="text"
                  value={deleteConfirmText}
                  onChange={(e) => setDeleteConfirmText(e.target.value)}
                  placeholder="DELETE"
                />
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowDeleteTenantModal(false)}>
                Cancel
              </button>
              <button
                className="btn btn-danger"
                onClick={handleDeleteTenant}
                disabled={deleteConfirmText !== 'DELETE'}
              >
                Delete Organization
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
