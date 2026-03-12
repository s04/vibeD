import { useState, useEffect, useCallback } from 'react'
import {
  ArtifactSummary,
  TargetInfo,
  fetchArtifacts,
  fetchTargets,
  deleteArtifact,
  fetchWhoami,
  fetchOrganization,
} from './api/client'
import ArtifactList from './components/ArtifactList'
import DeploymentTargets from './components/DeploymentTargets'
import LogViewer from './components/LogViewer'
import VersionHistory from './components/VersionHistory'
import ShareDialog from './components/ShareDialog'
import SetupGuide from './components/SetupGuide'
import './App.css'

function App() {
  const [artifacts, setArtifacts] = useState<ArtifactSummary[]>([])
  const [targets, setTargets] = useState<TargetInfo[]>([])
  const [selectedArtifactId, setSelectedArtifactId] = useState<string | null>(null)
  const [versionArtifactId, setVersionArtifactId] = useState<string | null>(null)
  const [shareArtifactId, setShareArtifactId] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [currentUser, setCurrentUser] = useState<string>('')
  const [isAdmin, setIsAdmin] = useState(false)
  const [orgName, setOrgName] = useState<string>('')

  // Fetch user identity and org info on mount
  useEffect(() => {
    fetchWhoami()
      .then((info) => {
        setCurrentUser(info.user_id)
        setIsAdmin(info.role === 'admin')
      })
      .catch(() => {
        // Auth may be disabled — that's fine
      })

    fetchOrganization()
      .then((org) => setOrgName(org.name))
      .catch(() => {
        // Organization may not be configured
      })
  }, [])

  const loadData = useCallback(async () => {
    try {
      setError(null)
      const [arts, tgts] = await Promise.all([fetchArtifacts(), fetchTargets()])
      setArtifacts(arts)
      setTargets(tgts)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data')
    } finally {
      setLoading(false)
    }
  }, [])

  const handleDelete = useCallback(async (id: string) => {
    await deleteArtifact(id)
    setArtifacts((prev) => prev.filter((a) => a.id !== id))
  }, [])

  useEffect(() => {
    loadData()
    const interval = setInterval(loadData, 5000) // Poll every 5s
    return () => clearInterval(interval)
  }, [loadData])

  return (
    <div className="app">
      <header className="header">
        <div className="header-left">
          <h1 className="logo">
            <img src="/logo.png" alt="vibeD" className="logo-img" />
            vibeD
          </h1>
          <span className="subtitle">Workload Orchestrator</span>
          {orgName && <span className="org-badge">{orgName}</span>}
        </div>
        <div className="header-right">
          {currentUser && (
            <span className="user-info">
              {isAdmin && <span className="admin-badge">admin</span>}
              {currentUser}
            </span>
          )}
          <button className="refresh-btn" onClick={loadData} disabled={loading}>
            {loading ? 'Loading...' : 'Refresh'}
          </button>
        </div>
      </header>

      {error && (
        <div className="error-banner">
          {error}
          <button onClick={() => setError(null)}>Dismiss</button>
        </div>
      )}

      <main className="main">
        <section className="section">
          <SetupGuide />
        </section>

        <section className="section">
          <DeploymentTargets targets={targets} />
        </section>

        <section className="section">
          <h2 className="section-title">
            Deployed Artifacts
            <span className="count">{artifacts.length}</span>
          </h2>
          <ArtifactList
            artifacts={artifacts}
            currentUser={currentUser}
            isAdmin={isAdmin}
            onViewLogs={(id) => setSelectedArtifactId(id)}
            onViewVersions={(id) => setVersionArtifactId(id)}
            onShare={(id) => setShareArtifactId(id)}
            onDelete={handleDelete}
          />
        </section>
      </main>

      {selectedArtifactId && (
        <LogViewer
          artifactId={selectedArtifactId}
          onClose={() => setSelectedArtifactId(null)}
        />
      )}

      {versionArtifactId && (
        <VersionHistory
          artifactId={versionArtifactId}
          onClose={() => setVersionArtifactId(null)}
          onRollbackComplete={loadData}
        />
      )}

      {shareArtifactId && (
        <ShareDialog
          artifactId={shareArtifactId}
          onClose={() => setShareArtifactId(null)}
          onShareComplete={loadData}
        />
      )}
    </div>
  )
}

export default App
