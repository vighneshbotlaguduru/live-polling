import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { polls as pollsApi } from '../services/api';
import useLiveResults from '../hooks/useLiveResults';
import LiveChart from '../components/LiveChart';
import ShareModal from '../components/ShareModal';
import LoadingSpinner from '../components/LoadingSpinner';

export default function VotePage() {
  const { shareCode } = useParams();

  const [poll, setPoll] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedOption, setSelectedOption] = useState(null);
  const [hasVoted, setHasVoted] = useState(false);
  const [voting, setVoting] = useState(false);
  const [voteError, setVoteError] = useState('');
  const [showShare, setShowShare] = useState(false);

  // SSE hook — only active after voting or if poll is closed
  const { counts, connected } = useLiveResults(
    hasVoted || (poll && !poll.is_active) ? poll?.id : null
  );

  // Fetch poll data
  useEffect(() => {
    const fetchPoll = async () => {
      try {
        const data = await pollsApi.get(shareCode);
        setPoll(data);

        // Check if user already voted (localStorage)
        const votedKey = `voted_${data.id}`;
        if (localStorage.getItem(votedKey)) {
          setHasVoted(true);
        }
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };
    fetchPoll();
  }, [shareCode]);

  const handleVote = async () => {
    if (!selectedOption || voting) return;
    setVoteError('');
    setVoting(true);

    try {
      await pollsApi.vote(poll.id, selectedOption);
      localStorage.setItem(`voted_${poll.id}`, selectedOption);
      setHasVoted(true);
    } catch (err) {
      setVoteError(err.message);
    } finally {
      setVoting(false);
    }
  };

  if (loading) return <LoadingSpinner fullPage />;

  if (error) {
    return (
      <div className="page-container">
        <div className="vote-page">
          <div className="glass-card vote-card" style={{ textAlign: 'center' }}>
            <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>😕</div>
            <h2 style={{ marginBottom: '0.5rem' }}>Poll Not Found</h2>
            <p style={{ color: 'var(--text-secondary)' }}>{error}</p>
          </div>
        </div>
      </div>
    );
  }

  const isActive = poll.is_active && !(poll.expires_at && new Date(poll.expires_at) < new Date());

  // Use live counts from SSE if available, otherwise fall back to poll data
  const displayCounts =
    Object.keys(counts).length > 0
      ? counts
      : poll.options.reduce((acc, opt) => {
          acc[opt.id] = opt.vote_count;
          return acc;
        }, {});

  return (
    <div className="page-container">
      <div className="vote-page">
        <div className="glass-card vote-card" id="vote-card">
          {/* Poll question */}
          <h1 className="vote-question" id="vote-question">
            {poll.question}
          </h1>

          <div className="vote-info">
            <span>{poll.options.length} options</span>
            {!isActive && (
              <span style={{ color: 'var(--warning)' }}>
                {poll.expires_at && new Date(poll.expires_at) < new Date()
                  ? '⏰ Expired'
                  : '🔒 Closed'}
              </span>
            )}
          </div>

          {/* Closed/Expired banner */}
          {!isActive && !hasVoted && (
            <div className="poll-closed-banner">
              ⚠️ This poll is no longer accepting votes.
            </div>
          )}

          {/* Voting interface */}
          {!hasVoted && isActive ? (
            <>
              <div className="vote-options" id="vote-options">
                {poll.options.map((option) => (
                  <button
                    key={option.id}
                    className={`vote-option ${
                      selectedOption === option.id ? 'selected' : ''
                    }`}
                    onClick={() => setSelectedOption(option.id)}
                    disabled={voting}
                    id={`vote-option-${option.id}`}
                  >
                    {option.text}
                  </button>
                ))}
              </div>

              {voteError && (
                <div className="form-error" id="vote-error">
                  {voteError}
                </div>
              )}

              <div className="vote-submit-area">
                <button
                  className="btn btn-primary btn-full btn-lg"
                  onClick={handleVote}
                  disabled={!selectedOption || voting}
                  id="submit-vote-btn"
                >
                  {voting ? 'Submitting...' : 'Cast Your Vote'}
                </button>
              </div>
            </>
          ) : (
            <>
              {/* Live results */}
              <LiveChart
                options={poll.options}
                counts={displayCounts}
                connected={connected}
              />

              {/* Share button */}
              <div style={{ marginTop: '1.5rem', textAlign: 'center' }}>
                <button
                  className="btn btn-secondary"
                  onClick={() => setShowShare(true)}
                  id="share-results-btn"
                >
                  🔗 Share This Poll
                </button>
              </div>
            </>
          )}
        </div>
      </div>

      {showShare && (
        <ShareModal shareCode={shareCode} onClose={() => setShowShare(false)} />
      )}
    </div>
  );
}
