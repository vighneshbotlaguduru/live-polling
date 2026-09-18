import { useState, useEffect, useRef, useCallback } from 'react';
import { getLiveURL } from '../services/api';

/**
 * Custom hook that connects to the SSE endpoint for a given poll
 * and returns live-updating vote counts.
 *
 * @param {string} pollId - The MongoDB ObjectID of the poll.
 * @returns {{ counts: Object, connected: boolean }}
 */
export default function useLiveResults(pollId) {
  const [counts, setCounts] = useState({});
  const [connected, setConnected] = useState(false);
  const esRef = useRef(null);
  const reconnectTimer = useRef(null);

  const connect = useCallback(() => {
    if (!pollId) return;

    // Close existing connection
    if (esRef.current) {
      esRef.current.close();
    }

    const url = getLiveURL(pollId);
    const es = new EventSource(url);
    esRef.current = es;

    es.addEventListener('init', (e) => {
      try {
        setCounts(JSON.parse(e.data));
        setConnected(true);
      } catch (err) {
        console.error('Failed to parse init data:', err);
      }
    });

    es.addEventListener('vote', (e) => {
      try {
        setCounts(JSON.parse(e.data));
      } catch (err) {
        console.error('Failed to parse vote data:', err);
      }
    });

    es.onerror = () => {
      es.close();
      setConnected(false);
      // Auto-reconnect after 3 seconds
      reconnectTimer.current = setTimeout(connect, 3000);
    };
  }, [pollId]);

  useEffect(() => {
    connect();

    return () => {
      if (esRef.current) {
        esRef.current.close();
      }
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
      }
    };
  }, [connect]);

  return { counts, connected };
}
