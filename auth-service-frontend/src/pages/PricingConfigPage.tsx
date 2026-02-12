import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Navbar from '../components/Navbar'
import { useUserSync, hasPermission } from '../hooks/useUserSync'
import { usePricingConfig } from '../hooks/usePricingConfig'
import type {
  PricingConfig,
  CloudProvider,
  PricingTier,
  ResourceType,
  PricingConfigFormData,
  PricingRateFormData,
} from '../types/pricing'
import {
  providerDisplayNames,
  tierDisplayNames,
  resourceDisplayNames,
  defaultUnits,
} from '../types/pricing'
import './PricingConfigPage.css'

export default function PricingConfigPage() {
  const navigate = useNavigate()
  const { role, isSynced } = useUserSync()
  const {
    configs,
    presets,
    loading,
    error: fetchError,
    refresh,
    createConfig,
    updateConfig,
    deleteConfig,
    addRate,
    deleteRate,
    importProviderDefaults,
  } = usePricingConfig()

  // UI State
  const [selectedConfig, setSelectedConfig] = useState<PricingConfig | null>(null)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showImportModal, setShowImportModal] = useState(false)
  const [showAddRateModal, setShowAddRateModal] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState<number | null>(null)

  // Messages
  const [successMessage, setSuccessMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Form State
  const [configForm, setConfigForm] = useState<PricingConfigFormData>({
    name: '',
    provider: 'aws',
    region: '',
    is_default: false,
  })

  const [rateForm, setRateForm] = useState<PricingRateFormData>({
    resource_type: 'cpu',
    pricing_tier: 'on_demand',
    instance_family: '',
    unit: 'core-hour',
    cost_per_unit: 0,
  })

  const [importForm, setImportForm] = useState({
    provider: 'aws' as CloudProvider,
    name: '',
    region: '',
  })

  // Check permissions
  const isAdmin = hasPermission(role, 'admin')

  // Message helpers
  const showSuccess = (message: string) => {
    setSuccessMessage(message)
    setTimeout(() => setSuccessMessage(null), 3000)
  }

  const showError = (message: string) => {
    setError(message)
    setTimeout(() => setError(null), 5000)
  }

  // Handlers
  const handleCreateConfig = async () => {
    if (!configForm.name.trim()) {
      showError('Please enter a configuration name')
      return
    }

    try {
      await createConfig(configForm)
      showSuccess('Pricing configuration created successfully')
      setShowCreateModal(false)
      setConfigForm({ name: '', provider: 'aws', region: '', is_default: false })
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to create configuration')
    }
  }

  const handleImportDefaults = async () => {
    if (!importForm.name.trim()) {
      showError('Please enter a configuration name')
      return
    }

    try {
      await importProviderDefaults(importForm.provider, importForm.name, importForm.region)
      showSuccess(`${providerDisplayNames[importForm.provider]} default pricing imported successfully`)
      setShowImportModal(false)
      setImportForm({ provider: 'aws', name: '', region: '' })
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to import defaults')
    }
  }

  const handleSetDefault = async (config: PricingConfig) => {
    try {
      await updateConfig(config.id, { is_default: true })
      showSuccess(`${config.name} is now the default configuration`)
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to set default')
    }
  }

  const handleDeleteConfig = async (id: number) => {
    try {
      await deleteConfig(id)
      showSuccess('Configuration deleted successfully')
      setShowDeleteConfirm(null)
      if (selectedConfig?.id === id) {
        setSelectedConfig(null)
      }
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to delete configuration')
    }
  }

  const handleAddRate = async () => {
    if (!selectedConfig) return
    if (rateForm.cost_per_unit <= 0) {
      showError('Cost per unit must be greater than 0')
      return
    }

    try {
      await addRate(selectedConfig.id, rateForm)
      showSuccess('Rate added successfully')
      setShowAddRateModal(false)
      setRateForm({
        resource_type: 'cpu',
        pricing_tier: 'on_demand',
        instance_family: '',
        unit: 'core-hour',
        cost_per_unit: 0,
      })
      // Refresh to get updated config with new rate
      await refresh()
      const updated = configs.find(c => c.id === selectedConfig.id)
      if (updated) setSelectedConfig(updated)
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to add rate')
    }
  }

  const handleDeleteRate = async (rateId: number) => {
    try {
      await deleteRate(rateId)
      showSuccess('Rate deleted successfully')
      await refresh()
      if (selectedConfig) {
        const updated = configs.find(c => c.id === selectedConfig.id)
        if (updated) setSelectedConfig(updated)
      }
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to delete rate')
    }
  }

  const handleResourceTypeChange = (type: ResourceType) => {
    setRateForm({
      ...rateForm,
      resource_type: type,
      unit: defaultUnits[type],
    })
  }

  const formatCurrency = (value: number) => {
    if (value < 0.01) {
      return `$${value.toFixed(6)}`
    }
    return `$${value.toFixed(4)}`
  }

  // Loading state
  if (!isSynced || loading) {
    return (
      <div className="page-container">
        <Navbar />
        <div className="page-content">
          <div className="loading-container">
            <div className="loading-spinner"></div>
            <p>Loading pricing configurations...</p>
          </div>
        </div>
      </div>
    )
  }

  // Permission check
  if (!isAdmin) {
    return (
      <div className="page-container">
        <Navbar />
        <div className="page-content">
          <div className="pricing-config-container">
            <div className="permission-denied">
              <h2>Access Denied</h2>
              <p>You need admin permissions to manage pricing configurations.</p>
              <button className="btn btn-primary" onClick={() => navigate('/dashboard')}>
                Go to Dashboard
              </button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="page-container">
      <Navbar />
      <div className="page-content">
        <div className="pricing-config-container">
          {/* Header */}
          <div className="pricing-header">
            <div>
              <h1>Cloud Pricing Configuration</h1>
              <p>Manage pricing rates for accurate cost calculations across your clusters</p>
            </div>
            <div className="header-actions">
              <button className="btn btn-secondary" onClick={() => setShowImportModal(true)}>
                Import Defaults
              </button>
              <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
                Create Config
              </button>
            </div>
          </div>

          {/* Messages */}
          {successMessage && (
            <div className="message success">{successMessage}</div>
          )}
          {(error || fetchError) && (
            <div className="message error">{error || fetchError}</div>
          )}

          {/* Main Content */}
          <div className="pricing-content">
            {/* Configs List */}
            <div className="configs-panel">
              <h2>Pricing Configurations</h2>
              {configs.length === 0 ? (
                <div className="empty-state">
                  <p>No pricing configurations yet.</p>
                  <p>Create one or import cloud provider defaults.</p>
                </div>
              ) : (
                <div className="configs-list">
                  {configs.map(config => (
                    <div
                      key={config.id}
                      className={`config-card ${selectedConfig?.id === config.id ? 'selected' : ''}`}
                      onClick={() => setSelectedConfig(config)}
                    >
                      <div className="config-header">
                        <span className={`provider-badge ${config.provider}`}>
                          {config.provider.toUpperCase()}
                        </span>
                        {config.is_default && (
                          <span className="default-badge">Default</span>
                        )}
                      </div>
                      <h3>{config.name}</h3>
                      {config.region && <p className="config-region">{config.region}</p>}
                      <p className="config-rates">
                        {config.rates?.length || 0} rate{(config.rates?.length || 0) !== 1 ? 's' : ''}
                      </p>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Config Details */}
            <div className="details-panel">
              {selectedConfig ? (
                <>
                  <div className="details-header">
                    <div>
                      <h2>{selectedConfig.name}</h2>
                      <p>
                        <span className={`provider-badge ${selectedConfig.provider}`}>
                          {providerDisplayNames[selectedConfig.provider]}
                        </span>
                        {selectedConfig.region && <span className="region-text"> - {selectedConfig.region}</span>}
                      </p>
                    </div>
                    <div className="details-actions">
                      {!selectedConfig.is_default && (
                        <button
                          className="btn btn-secondary"
                          onClick={() => handleSetDefault(selectedConfig)}
                        >
                          Set as Default
                        </button>
                      )}
                      <button
                        className="btn btn-danger"
                        onClick={() => setShowDeleteConfirm(selectedConfig.id)}
                      >
                        Delete
                      </button>
                    </div>
                  </div>

                  {/* Rates Table */}
                  <div className="rates-section">
                    <div className="section-header">
                      <h3>Pricing Rates</h3>
                      <button
                        className="btn btn-primary btn-small"
                        onClick={() => setShowAddRateModal(true)}
                      >
                        Add Rate
                      </button>
                    </div>

                    {(!selectedConfig.rates || selectedConfig.rates.length === 0) ? (
                      <div className="empty-state">
                        <p>No rates configured. Add rates to define pricing.</p>
                      </div>
                    ) : (
                      <table className="rates-table">
                        <thead>
                          <tr>
                            <th>Resource</th>
                            <th>Tier</th>
                            <th>Instance Family</th>
                            <th>Unit</th>
                            <th>Cost</th>
                            <th>Actions</th>
                          </tr>
                        </thead>
                        <tbody>
                          {selectedConfig.rates.map(rate => (
                            <tr key={rate.id}>
                              <td>{resourceDisplayNames[rate.resource_type]}</td>
                              <td>{tierDisplayNames[rate.pricing_tier]}</td>
                              <td>{rate.instance_family || '—'}</td>
                              <td>{rate.unit}</td>
                              <td className="cost-cell">{formatCurrency(rate.cost_per_unit)}</td>
                              <td>
                                <button
                                  className="btn btn-danger btn-small"
                                  onClick={() => handleDeleteRate(rate.id)}
                                >
                                  Delete
                                </button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    )}
                  </div>

                  {/* Provider Preset Reference */}
                  {presets && presets.presets[selectedConfig.provider] && (
                    <div className="presets-reference">
                      <h3>Provider Default Rates Reference</h3>
                      <p className="help-text">These are the default rates for {providerDisplayNames[selectedConfig.provider]}. Use them as a reference when adding rates.</p>
                      <div className="preset-rates">
                        {Object.entries(presets.presets[selectedConfig.provider]).map(([key, value]) => (
                          <div key={key} className="preset-rate">
                            <span className="preset-key">{key}</span>
                            <span className="preset-value">{formatCurrency(value)}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <div className="no-selection">
                  <h3>Select a Configuration</h3>
                  <p>Click on a configuration from the list to view and manage its rates.</p>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Create Config Modal */}
        {showCreateModal && (
          <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
            <div className="modal-content" onClick={e => e.stopPropagation()}>
              <h2>Create Pricing Configuration</h2>
              <div className="form-group">
                <label>Name</label>
                <input
                  type="text"
                  value={configForm.name}
                  onChange={e => setConfigForm({ ...configForm, name: e.target.value })}
                  placeholder="e.g., AWS Production"
                />
              </div>
              <div className="form-group">
                <label>Cloud Provider</label>
                <select
                  value={configForm.provider}
                  onChange={e => setConfigForm({ ...configForm, provider: e.target.value as CloudProvider })}
                >
                  <option value="aws">Amazon Web Services (AWS)</option>
                  <option value="gcp">Google Cloud Platform (GCP)</option>
                  <option value="azure">Microsoft Azure</option>
                  <option value="oci">Oracle Cloud Infrastructure (OCI)</option>
                  <option value="custom">Custom</option>
                </select>
              </div>
              <div className="form-group">
                <label>Region (optional)</label>
                <input
                  type="text"
                  value={configForm.region}
                  onChange={e => setConfigForm({ ...configForm, region: e.target.value })}
                  placeholder="e.g., us-east-1"
                />
              </div>
              <div className="form-group checkbox">
                <label>
                  <input
                    type="checkbox"
                    checked={configForm.is_default}
                    onChange={e => setConfigForm({ ...configForm, is_default: e.target.checked })}
                  />
                  Set as default configuration
                </label>
              </div>
              <div className="modal-actions">
                <button className="btn btn-secondary" onClick={() => setShowCreateModal(false)}>
                  Cancel
                </button>
                <button className="btn btn-primary" onClick={handleCreateConfig}>
                  Create
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Import Defaults Modal */}
        {showImportModal && (
          <div className="modal-overlay" onClick={() => setShowImportModal(false)}>
            <div className="modal-content" onClick={e => e.stopPropagation()}>
              <h2>Import Provider Defaults</h2>
              <p className="modal-description">
                Import default pricing rates from a cloud provider. This creates a new configuration with pre-configured rates.
              </p>
              <div className="form-group">
                <label>Cloud Provider</label>
                <select
                  value={importForm.provider}
                  onChange={e => setImportForm({ ...importForm, provider: e.target.value as CloudProvider })}
                >
                  <option value="aws">Amazon Web Services (AWS)</option>
                  <option value="gcp">Google Cloud Platform (GCP)</option>
                  <option value="azure">Microsoft Azure</option>
                  <option value="oci">Oracle Cloud Infrastructure (OCI)</option>
                </select>
              </div>
              <div className="form-group">
                <label>Configuration Name</label>
                <input
                  type="text"
                  value={importForm.name}
                  onChange={e => setImportForm({ ...importForm, name: e.target.value })}
                  placeholder={`e.g., ${importForm.provider.toUpperCase()} Default`}
                />
              </div>
              <div className="form-group">
                <label>Region (optional)</label>
                <input
                  type="text"
                  value={importForm.region}
                  onChange={e => setImportForm({ ...importForm, region: e.target.value })}
                  placeholder="e.g., us-east-1"
                />
              </div>
              <div className="modal-actions">
                <button className="btn btn-secondary" onClick={() => setShowImportModal(false)}>
                  Cancel
                </button>
                <button className="btn btn-primary" onClick={handleImportDefaults}>
                  Import
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Add Rate Modal */}
        {showAddRateModal && (
          <div className="modal-overlay" onClick={() => setShowAddRateModal(false)}>
            <div className="modal-content" onClick={e => e.stopPropagation()}>
              <h2>Add Pricing Rate</h2>
              <div className="form-group">
                <label>Resource Type</label>
                <select
                  value={rateForm.resource_type}
                  onChange={e => handleResourceTypeChange(e.target.value as ResourceType)}
                >
                  <option value="cpu">CPU</option>
                  <option value="memory">Memory</option>
                  <option value="gpu">GPU</option>
                  <option value="storage">Storage</option>
                  <option value="network">Network</option>
                </select>
              </div>
              <div className="form-group">
                <label>Pricing Tier</label>
                <select
                  value={rateForm.pricing_tier}
                  onChange={e => setRateForm({ ...rateForm, pricing_tier: e.target.value as PricingTier })}
                >
                  <option value="on_demand">On-Demand</option>
                  <option value="spot">Spot</option>
                  <option value="preemptible">Preemptible</option>
                  <option value="reserved_1yr">Reserved (1 Year)</option>
                  <option value="reserved_3yr">Reserved (3 Year)</option>
                </select>
              </div>
              <div className="form-group">
                <label>Instance Family (optional)</label>
                <input
                  type="text"
                  value={rateForm.instance_family}
                  onChange={e => setRateForm({ ...rateForm, instance_family: e.target.value })}
                  placeholder="e.g., m5, c5, n1-standard"
                />
                <span className="help-text">Leave blank for generic pricing across all instance types</span>
              </div>
              <div className="form-group">
                <label>Unit</label>
                <input
                  type="text"
                  value={rateForm.unit}
                  onChange={e => setRateForm({ ...rateForm, unit: e.target.value })}
                  placeholder="e.g., core-hour, gb-hour"
                />
              </div>
              <div className="form-group">
                <label>Cost per Unit ($)</label>
                <input
                  type="number"
                  step="0.000001"
                  min="0"
                  value={rateForm.cost_per_unit}
                  onChange={e => setRateForm({ ...rateForm, cost_per_unit: parseFloat(e.target.value) || 0 })}
                  placeholder="0.0425"
                />
              </div>
              <div className="modal-actions">
                <button className="btn btn-secondary" onClick={() => setShowAddRateModal(false)}>
                  Cancel
                </button>
                <button className="btn btn-primary" onClick={handleAddRate}>
                  Add Rate
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Delete Confirmation Modal */}
        {showDeleteConfirm !== null && (
          <div className="modal-overlay" onClick={() => setShowDeleteConfirm(null)}>
            <div className="modal-content modal-small" onClick={e => e.stopPropagation()}>
              <h2>Delete Configuration</h2>
              <p>Are you sure you want to delete this pricing configuration? This action cannot be undone.</p>
              <div className="modal-actions">
                <button className="btn btn-secondary" onClick={() => setShowDeleteConfirm(null)}>
                  Cancel
                </button>
                <button className="btn btn-danger" onClick={() => handleDeleteConfig(showDeleteConfirm)}>
                  Delete
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
