import '../App.css'
import './AutoSignOutWarning.css'

interface AutoSignOutWarningProps {
  timeRemaining: number
  onContinue: () => void
  onSignOut: () => void
}

export default function AutoSignOutWarning({
  timeRemaining,
  onContinue,
  onSignOut,
}: AutoSignOutWarningProps) {
  const minutes = Math.floor(timeRemaining / 60)
  const seconds = timeRemaining % 60

  // Format time as MM:SS
  const formatTime = (mins: number, secs: number) => {
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }

  return (
    <div className="auto-signout-overlay">
      <div className="auto-signout-modal">
        <div className="auto-signout-header">
          <h2>Session Timeout Warning</h2>
        </div>
        <div className="auto-signout-body">
          <div className="auto-signout-icon">⏰</div>
          <p className="auto-signout-message">
            Your session will expire in <strong>{formatTime(minutes, seconds)}</strong>
          </p>
          <p className="auto-signout-submessage">
            Would you like to continue your session or sign out now?
          </p>
        </div>
        <div className="auto-signout-footer">
          <button
            className="btn btn-secondary auto-signout-btn"
            onClick={onSignOut}
          >
            Sign Out
          </button>
          <button
            className="btn btn-primary auto-signout-btn"
            onClick={onContinue}
          >
            Continue Session
          </button>
        </div>
      </div>
    </div>
  )
}
