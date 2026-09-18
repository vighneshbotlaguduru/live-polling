import { useMemo } from 'react';

/**
 * LiveChart renders animated horizontal bars for real-time vote results.
 *
 * @param {Array} options - Poll options with { id, text }
 * @param {Object} counts - Vote counts map { optionId: count }
 */
export default function LiveChart({ options, counts, connected }) {
  const totalVotes = useMemo(() => {
    return Object.values(counts).reduce((sum, c) => sum + c, 0);
  }, [counts]);

  const maxVotes = useMemo(() => {
    return Math.max(...Object.values(counts), 0);
  }, [counts]);

  return (
    <div className="live-chart" id="live-chart">
      <div className="results-header">
        <div className="results-total">
          <strong>{totalVotes}</strong> vote{totalVotes !== 1 ? 's' : ''}
        </div>
        {connected && (
          <div className="live-badge" id="live-badge">
            <span className="live-badge-dot"></span>
            LIVE
          </div>
        )}
      </div>

      <div className="result-bars">
        {options.map((option) => {
          const count = counts[option.id] || 0;
          const percentage = totalVotes > 0 ? (count / totalVotes) * 100 : 0;
          const isWinner = count === maxVotes && count > 0;

          return (
            <div
              key={option.id}
              className={`result-bar-item ${isWinner ? 'result-bar-winner' : ''}`}
            >
              <div className="result-bar-label">
                <span className="result-bar-text">{option.text}</span>
                <div className="result-bar-stats">
                  <span className="result-bar-count">
                    {count} vote{count !== 1 ? 's' : ''}
                  </span>
                  <span className="result-bar-percent">
                    {percentage.toFixed(1)}%
                  </span>
                </div>
              </div>
              <div className="result-bar-track">
                <div
                  className="result-bar-fill"
                  style={{ width: `${Math.max(percentage, 0.5)}%` }}
                />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
