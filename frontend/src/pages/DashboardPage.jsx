import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { polls as pollsApi } from '../services/api';
import PollCard from '../components/PollCard';
import LoadingSpinner from '../components/LoadingSpinner';

export default function DashboardPage() {
  const [polls, setPolls] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchPolls = useCallback(async () => {
    try {
      const data = await pollsApi.list();
      setPolls(data);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchPolls();
  }, [fetchPolls]);

  const handleClose = async (pollId) => {
    if (!window.confirm('Close this poll? Voting will be disabled.')) return;
    try {
      await pollsApi.close(pollId);
      fetchPolls();
    } catch (err) {
      alert(err.message);
    }
  };

  const handleDelete = async (pollId) => {
    if (!window.confirm('Delete this poll? This cannot be undone.')) return;
    try {
      await pollsApi.remove(pollId);
      setPolls((prev) => prev.filter((p) => p.id !== pollId));
    } catch (err) {
      alert(err.message);
    }
  };

  if (loading) return <LoadingSpinner fullPage />;

  return (
    <div className="page-container">
      <div className="content-container">
        <div className="dashboard-header">
          <h1 className="dashboard-title" id="dashboard-title">Your Polls</h1>
          <Link to="/create" className="btn btn-primary" id="create-poll-btn">
            + Create New Poll
          </Link>
        </div>

        {error && <div className="form-error">{error}</div>}

        {polls.length === 0 ? (
          <div className="empty-state" id="empty-state">
            <div className="empty-state-icon">📊</div>
            <p className="empty-state-text">
              You haven't created any polls yet.
            </p>
            <Link to="/create" className="btn btn-primary">
              Create Your First Poll
            </Link>
          </div>
        ) : (
          <div className="polls-grid" id="polls-grid">
            {polls.map((poll) => (
              <PollCard
                key={poll.id}
                poll={poll}
                onClose={handleClose}
                onDelete={handleDelete}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
