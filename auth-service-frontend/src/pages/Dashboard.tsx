import { useState, useEffect } from 'react'
import { useUser } from '@clerk/clerk-react'
import Navbar from '../components/Navbar'
import '../App.css'

const API_SERVER_URL = import.meta.env.VITE_API_SERVER_URL || 'http://localhost:8080'

interface APIKey {
  key_id: string
  secret: string
  full_key: string
  created_at: string
}

export default function Dashboard() {
  const { isLoaded } = useUser()
  const [apiKey, setApiKey] = useState<APIKey | null>(null)
  const [isGenerating, setIsGenerating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showKey, setShowKey] = useState(false)
  const [copied, setCopied] = useState(false)
  const [formData, setFormData] = useState({
    expires_at: '',
  })

  useEffect(() => {
    // Load API key from localStorage on mount
    const savedKey = localStorage.getItem('api_key')
    if (savedKey) {
      try {
        setApiKey(JSON.parse(savedKey))
      } catch (e) {
        console.error('Failed to parse saved API key:', e)
      }
    }
  }, [])

  const handleGenerateKey = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsGenerating(true)
    setError(null)

    try {
      interface CreateKeyPayload {
        tenant_id: number
        scopes: string[]
        expires_at?: string
      }

      // TODO: Get tenant_id from user metadata once tenant assignment is implemented
      // For now, using default tenant_id of 1
      const payload: CreateKeyPayload = {
        tenant_id: 1,
        scopes: [],
      }

      if (formData.expires_at) {
        payload.expires_at = new Date(formData.expires_at).toISOString()
      }

      const response = await fetch(`${API_SERVER_URL}/v1/admin/api_keys`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      })

      if (!response.ok) {
        let errorMessage = 'Failed to create API key'
        try {
          const errorData = await response.json()
          errorMessage = errorData.error || errorMessage
        } catch {
          errorMessage = `HTTP ${response.status}: ${response.statusText}`
        }
        throw new Error(errorMessage)
      }

      const data = await response.json()
      const newKey: APIKey = {
        key_id: data.key_id,
        secret: data.secret,
        full_key: `${data.key_id}:${data.secret}`,
        created_at: new Date().toISOString(),
      }

      setApiKey(newKey)
      localStorage.setItem('api_key', JSON.stringify(newKey))
      setShowKey(true)
      setCopied(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
      console.error('Error generating API key:', err)
    } finally {
      setIsGenerating(false)
    }
  }

  const handleCopyKey = () => {
    if (apiKey) {
      navigator.clipboard.writeText(apiKey.full_key)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const handleCopyKeyId = () => {
    if (apiKey) {
      navigator.clipboard.writeText(apiKey.key_id)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  if (!isLoaded) {
    return (
      <div className="page-container">
        <Navbar />
        <div className="page-content">
          <div className="page-card">
            <p>Loading...</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="page-container">
      <Navbar />
      <div className="page-content">
        <div className="dashboard-card">
          <h1>Dashboard</h1>

          {/* API Key Management Section */}
          <div className="api-key-section">
            <h2>API Key Management</h2>
            
            {/* Display Current API Key */}
            {apiKey && (
              <div className="api-key-display">
                <h3>Current API Key</h3>
                <div className="api-key-info">
                  <div className="api-key-field">
                    <label>Key ID:</label>
                    <div className="api-key-value">
                      <code>{apiKey.key_id}</code>
                      <button
                        type="button"
                        className="copy-button"
                        onClick={handleCopyKeyId}
                        title="Copy Key ID"
                      >
                        {copied ? '✓' : '📋'}
                      </button>
                    </div>
                  </div>
                  <div className="api-key-field">
                    <label>Full API Key:</label>
                    <div className="api-key-value">
                      <code className={showKey ? '' : 'hidden-key'}>
                        {showKey ? apiKey.full_key : '••••••••••••••••••••••••••••••••'}
                      </code>
                      <button
                        type="button"
                        className="copy-button"
                        onClick={handleCopyKey}
                        title="Copy Full API Key"
                      >
                        {copied ? '✓' : '📋'}
                      </button>
                      <button
                        type="button"
                        className="toggle-button"
                        onClick={() => setShowKey(!showKey)}
                        title={showKey ? 'Hide' : 'Show'}
                      >
                        {showKey ? '👁️' : '👁️‍🗨️'}
                      </button>
                    </div>
                    <small className="api-key-warning">
                      ⚠️ Store this securely. The secret will only be shown once.
                    </small>
                  </div>
                </div>
              </div>
            )}

            {/* Generate New API Key Form */}
            <div className="api-key-form">
              <h3>{apiKey ? 'Generate New API Key' : 'Generate API Key'}</h3>
              {error && <div className="error-message">{error}</div>}
              <form onSubmit={handleGenerateKey}>
                <div className="form-group">
                  <label htmlFor="expires_at">Expiration Date (optional)</label>
                  <input
                    type="datetime-local"
                    id="expires_at"
                    value={formData.expires_at}
                    onChange={(e) => setFormData({ ...formData, expires_at: e.target.value })}
                  />
                </div>
                <button
                  type="submit"
                  className="generate-button"
                  disabled={isGenerating}
                >
                  {isGenerating ? 'Generating...' : 'Generate API Key'}
                </button>
              </form>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

