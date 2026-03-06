import { useState, useEffect } from 'react'
import { ArtifactVersion, fetchVersions, rollbackArtifact } from '../api/client'
import './VersionHistory.css'

interface Props {
  artifactId: string
  onClose: () => void
  onRollbackComplete: () => void
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export default function VersionHistory({ artifactId, onClose, onRollbackComplete }: Props) {
  const [versions, setVersions] = useState<ArtifactVersion[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [rollbackTarget, setRollbackTarget] = useState<number | null>(null)
  const [rolling, setRolling] = useState(false)

  useEffect(() => {
    let mounted = true
    async function load() {
      try {
        setLoading(true)
        setError(null)
        const data = await fetchVersions(artifactId)
        if (mounted) setVersions(data)
      } catch (err) {
        if (mounted) setError(err instanceof Error ? err.message : 'Failed to load versions')
      } finally {
        if (mounted) setLoading(false)
      }
    }
    load()
    return () => { mounted = false }
  }, [artifactId])

  const handleRollback = async (version: number) => {
    setRolling(true)
    try {
      await rollbackArtifact(artifactId, version)
      setRollbackTarget(null)
      onRollbackComplete()
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Rollback failed')
      setRolling(false)
    }
  }

  const latestVersion = versions.length > 0 ? Math.max(...versions.map(v => v.version)) : 0

  return (
    <div className="vh-overlay" onClick={onClose}>
      <div className="vh-panel" onClick={(e) => e.stopPropagation()}>
        <div className="vh-header">
          <h3>Version History</h3>
          <button className="vh-close" onClick={onClose}>&times;</button>
        </div>

        <div className="vh-content">
          {loading && <div className="vh-loading">Loading versions...</div>}
          {error && <div className="vh-error">{error}</div>}
          {!loading && !error && versions.length === 0 && (
            <div className="vh-empty">No version history available</div>
          )}

          {versions.map((v) => (
            <div key={v.version_id} className={`vh-row ${v.version === latestVersion ? 'vh-row-current' : ''}`}>
              <div className="vh-row-left">
                <span className="vh-version">v{v.version}</span>
                {v.version === latestVersion && <span className="vh-current-tag">current</span>}
              </div>
              <div className="vh-row-center">
                <div className="vh-row-meta">
                  <span className="vh-date">{formatDate(v.created_at)}</span>
                  {v.created_by && <span className="vh-user">by {v.created_by}</span>}
                </div>
                <div className="vh-row-details">
                  <span className="vh-image" title={v.image_ref}>{v.image_ref}</span>
                  <span className={`vh-status vh-status-${v.status}`}>{v.status}</span>
                </div>
              </div>
              <div className="vh-row-right">
                {v.version !== latestVersion && (
                  rollbackTarget === v.version ? (
                    <div className="vh-confirm">
                      <span className="vh-confirm-text">Rollback?</span>
                      <button
                        className="vh-btn vh-btn-danger"
                        onClick={() => handleRollback(v.version)}
                        disabled={rolling}
                      >
                        {rolling ? '...' : 'Yes'}
                      </button>
                      <button
                        className="vh-btn"
                        onClick={() => setRollbackTarget(null)}
                        disabled={rolling}
                      >
                        No
                      </button>
                    </div>
                  ) : (
                    <button className="vh-btn" onClick={() => setRollbackTarget(v.version)}>
                      Rollback
                    </button>
                  )
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
