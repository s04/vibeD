import { useState, useEffect } from 'react'
import { fetchArtifact, shareArtifact, unshareArtifact, Artifact } from '../api/client'
import './ShareDialog.css'

interface Props {
  artifactId: string
  onClose: () => void
  onShareComplete: () => void
}

export default function ShareDialog({ artifactId, onClose, onShareComplete }: Props) {
  const [artifact, setArtifact] = useState<Artifact | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [inputValue, setInputValue] = useState('')
  const [sharing, setSharing] = useState(false)
  const [removing, setRemoving] = useState<string | null>(null)

  useEffect(() => {
    let mounted = true
    async function load() {
      try {
        setLoading(true)
        setError(null)
        const data = await fetchArtifact(artifactId)
        if (mounted) setArtifact(data)
      } catch (err) {
        if (mounted) setError(err instanceof Error ? err.message : 'Failed to load artifact')
      } finally {
        if (mounted) setLoading(false)
      }
    }
    load()
    return () => { mounted = false }
  }, [artifactId])

  const handleShare = async () => {
    const userIds = inputValue
      .split(',')
      .map((s) => s.trim())
      .filter((s) => s.length > 0)
    if (userIds.length === 0) return

    setSharing(true)
    setError(null)
    try {
      await shareArtifact(artifactId, userIds)
      setInputValue('')
      // Refresh artifact data
      const updated = await fetchArtifact(artifactId)
      setArtifact(updated)
      onShareComplete()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to share')
    } finally {
      setSharing(false)
    }
  }

  const handleRemove = async (userId: string) => {
    setRemoving(userId)
    setError(null)
    try {
      await unshareArtifact(artifactId, [userId])
      const updated = await fetchArtifact(artifactId)
      setArtifact(updated)
      onShareComplete()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove')
    } finally {
      setRemoving(null)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !sharing) {
      handleShare()
    }
  }

  const sharedWith = artifact?.shared_with ?? []

  return (
    <div className="sd-overlay" onClick={onClose}>
      <div className="sd-panel" onClick={(e) => e.stopPropagation()}>
        <div className="sd-header">
          <h3>Share Artifact</h3>
          <button className="sd-close" onClick={onClose}>&times;</button>
        </div>

        <div className="sd-content">
          {loading && <div className="sd-loading">Loading...</div>}
          {error && <div className="sd-error">{error}</div>}

          {!loading && artifact && (
            <>
              <div className="sd-artifact-name">{artifact.name}</div>

              {/* Current shares */}
              <div className="sd-section">
                <div className="sd-section-title">Shared with</div>
                {sharedWith.length === 0 ? (
                  <div className="sd-empty">Not shared with anyone</div>
                ) : (
                  <div className="sd-user-list">
                    {sharedWith.map((uid) => (
                      <div key={uid} className="sd-user-row">
                        <span className="sd-user-name">{uid}</span>
                        <span className="sd-user-perm">read-only</span>
                        <button
                          className="sd-remove-btn"
                          onClick={() => handleRemove(uid)}
                          disabled={removing === uid}
                        >
                          {removing === uid ? '...' : 'Remove'}
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Add shares */}
              <div className="sd-section">
                <div className="sd-section-title">Add users</div>
                <div className="sd-input-row">
                  <input
                    className="sd-input"
                    type="text"
                    placeholder="User IDs (comma-separated)"
                    value={inputValue}
                    onChange={(e) => setInputValue(e.target.value)}
                    onKeyDown={handleKeyDown}
                    disabled={sharing}
                  />
                  <button
                    className="sd-share-btn"
                    onClick={handleShare}
                    disabled={sharing || inputValue.trim().length === 0}
                  >
                    {sharing ? 'Sharing...' : 'Share'}
                  </button>
                </div>
                <p className="sd-hint">
                  Shared users get read-only access (view status, logs, URL).
                </p>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
