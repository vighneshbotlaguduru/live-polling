import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { polls as pollsApi } from '../services/api';

export default function CreatePollPage() {
  const navigate = useNavigate();

  const [question, setQuestion] = useState('');
  const [options, setOptions] = useState(['', '']);
  const [expiresIn, setExpiresIn] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const addOption = () => {
    if (options.length >= 10) return;
    setOptions([...options, '']);
  };

  const removeOption = (index) => {
    if (options.length <= 2) return;
    setOptions(options.filter((_, i) => i !== index));
  };

  const updateOption = (index, value) => {
    const updated = [...options];
    updated[index] = value;
    setOptions(updated);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');

    const trimmedQuestion = question.trim();
    const trimmedOptions = options.map((o) => o.trim()).filter((o) => o.length > 0);

    if (trimmedQuestion.length < 5) {
      setError('Question must be at least 5 characters');
      return;
    }
    if (trimmedOptions.length < 2) {
      setError('Add at least 2 options');
      return;
    }

    // Check for duplicates
    const unique = new Set(trimmedOptions.map((o) => o.toLowerCase()));
    if (unique.size !== trimmedOptions.length) {
      setError('Options must be unique');
      return;
    }

    setLoading(true);
    try {
      const payload = {
        question: trimmedQuestion,
        options: trimmedOptions,
      };
      if (expiresIn) {
        payload.expires_in = parseInt(expiresIn, 10);
      }

      const poll = await pollsApi.create(payload);
      navigate(`/poll/${poll.share_code}`);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="page-container">
      <div className="content-container create-poll-page">
        <div className="glass-card create-poll-card">
          <div className="auth-header">
            <h1 className="auth-title">Create a Poll</h1>
            <p className="auth-subtitle">
              Ask a question, add options, and share with your audience.
            </p>
          </div>

          {error && <div className="form-error" id="create-error">{error}</div>}

          <form onSubmit={handleSubmit} id="create-poll-form">
            <div className="form-group">
              <label className="form-label" htmlFor="poll-question">
                Question
              </label>
              <textarea
                id="poll-question"
                className="form-input"
                placeholder="What would you like to ask?"
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                disabled={loading}
                rows={3}
              />
            </div>

            <div className="form-group">
              <label className="form-label">Options</label>
              <div className="option-list">
                {options.map((opt, i) => (
                  <div className="option-row" key={i}>
                    <span className="option-number">{i + 1}</span>
                    <input
                      className="form-input"
                      placeholder={`Option ${i + 1}`}
                      value={opt}
                      onChange={(e) => updateOption(i, e.target.value)}
                      disabled={loading}
                      id={`poll-option-${i}`}
                    />
                    {options.length > 2 && (
                      <button
                        type="button"
                        className="btn-remove-option"
                        onClick={() => removeOption(i)}
                        title="Remove option"
                      >
                        ✕
                      </button>
                    )}
                  </div>
                ))}
              </div>
              {options.length < 10 && (
                <button
                  type="button"
                  className="btn-add-option"
                  onClick={addOption}
                  id="add-option-btn"
                >
                  + Add option
                </button>
              )}
            </div>

            <div className="form-group">
              <label className="form-label" htmlFor="poll-expiry">
                Expiration (optional)
              </label>
              <select
                id="poll-expiry"
                className="expiry-select"
                value={expiresIn}
                onChange={(e) => setExpiresIn(e.target.value)}
                disabled={loading}
              >
                <option value="">No expiration</option>
                <option value="15">15 minutes</option>
                <option value="30">30 minutes</option>
                <option value="60">1 hour</option>
                <option value="360">6 hours</option>
                <option value="1440">24 hours</option>
                <option value="10080">1 week</option>
              </select>
            </div>

            <button
              type="submit"
              className="btn btn-primary btn-full btn-lg"
              disabled={loading}
              id="create-poll-submit"
            >
              {loading ? 'Creating...' : '🚀 Create Poll'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
