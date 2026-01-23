import { useState } from 'react'
import './APIKeyModal.css'

interface APIKeyModalProps {
  apiKey: string
  onClose: () => void
}

export default function APIKeyModal({ apiKey, onClose }: APIKeyModalProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(apiKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <div className="modal-header">
          <h2>Your API Key</h2>
        </div>

        <div className="modal-body">
          <div className="warning-box">
            <span className="warning-icon">!</span>
            <p>
              <strong>Important:</strong> This is the only time you will see this API key.
              Please copy and save it securely. You will need it to configure your cost-agent.
            </p>
          </div>

          <div className="api-key-container">
            <code className="api-key-value">{apiKey}</code>
            <button
              className={`copy-button ${copied ? 'copied' : ''}`}
              onClick={handleCopy}
            >
              {copied ? 'Copied!' : 'Copy'}
            </button>
          </div>

          <div className="usage-info">
            <h3>How to use this API key:</h3>
            <p>Set it as an environment variable for your cost-agent:</p>
            <code className="usage-code">AGENT_API_KEY={apiKey}</code>
            <p>Or create a Kubernetes secret:</p>
            <code className="usage-code">
              kubectl create secret generic cost-agent-api-key \<br />
              &nbsp;&nbsp;--from-literal=api-key={apiKey}
            </code>
          </div>
        </div>

        <div className="modal-footer">
          <button className="modal-button-primary" onClick={onClose}>
            I've saved my API key
          </button>
        </div>
      </div>
    </div>
  )
}
