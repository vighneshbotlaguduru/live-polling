import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import ShareModal from './ShareModal';

/**
 * PollCard displays a poll summary with status, stats, and action buttons.
 *
 * @param {Object} poll - The poll object
 * @param {function} onClose - Callback when poll is closed
 * @param {function} onDelete - Callback when poll is deleted
 */
export default function PollCard({ poll, onClose, onDelete }) {
  const [showShare, setShowShare] = useState(false);
  const navigate = useNavigate();

  const totalVotes = poll.options.reduce((sum, opt) => sum + opt.vote_count, 0);

  const getStatus = () => {
    if (poll.expires_at && new Date(poll.expires_at) < new Date()) {
      return { label: 'Expired', className: 'poll-status-expired' };
    }
    if (!poll.is_active) {
      return { label: 'Closed', className: 'poll-status-closed' };
    }
    return { label: 'Active', className: 'poll-status-active' };
  };

  const status = getStatus();

  const formatDate = (dateStr) => {
    return new Date(dateStr).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  };

  return (
    <>
      <div className="glass-card glass-card-interactive poll-card">
        <div className="poll-card-question">{poll.question}</div>

        <div className="poll-card-meta">
          <span className={`poll-status ${status.className}`}>
            <span className="poll-status-dot"></span>
            {status.label}
          </span>
          <span className="poll-card-stat">🗳️ {totalVotes} votes</span>
          <span className="poll-card-stat">📊 {poll.options.length} options</span>
        </div>

        <div style={{ fontSize: '0.75rem', color: 'var(--text-tertiary)' }}>
          Created {formatDate(poll.created_at)}
        </div>

        <div className="poll-card-actions">
          <button
            className="btn btn-secondary btn-sm"
            onClick={() => setShowShare(true)}
            id={`share-poll-${poll.id}`}
          >
            🔗 Share
          </button>
          <button
            className="btn btn-ghost btn-sm"
            onClick={() => navigate(`/poll/${poll.share_code}`)}
            id={`view-poll-${poll.id}`}
          >
            👁️ View
          </button>
          {poll.is_active && (
            <button
              className="btn btn-ghost btn-sm"
              onClick={() => onClose(poll.id)}
              id={`close-poll-${poll.id}`}
            >
              🔒 Close
            </button>
          )}
          <button
            className="btn btn-ghost btn-sm"
            onClick={() => onDelete(poll.id)}
            style={{ color: 'var(--danger)', marginLeft: 'auto' }}
            id={`delete-poll-${poll.id}`}
          >
            🗑️
          </button>
        </div>
      </div>

      {showShare && (
        <ShareModal shareCode={poll.share_code} onClose={() => setShowShare(false)} />
      )}
    </>
  );
}
