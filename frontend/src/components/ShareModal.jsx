import { useState } from 'react';

/**
 * ShareModal displays a shareable link and QR code for a poll.
 *
 * @param {string} shareCode - The poll's share code
 * @param {function} onClose - Callback to close the modal
 */
export default function ShareModal({ shareCode, onClose }) {
  const [copied, setCopied] = useState(false);

  const shareURL = `${window.location.origin}/poll/${shareCode}`;
  const qrURL = `https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(shareURL)}&bgcolor=ffffff&color=6366f1`;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(shareURL);
      setCopied(true);
      setTimeout(() => setCopied(false), 2500);
    } catch {
      // Fallback
      const input = document.getElementById('share-link-input');
      if (input) {
        input.select();
        document.execCommand('copy');
        setCopied(true);
        setTimeout(() => setCopied(false), 2500);
      }
    }
  };

  const handleOverlayClick = (e) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return (
    <>
      <div className="modal-overlay" onClick={handleOverlayClick} id="share-modal">
        <div className="modal-content glass-card">
          <div className="modal-header">
            <h2 className="modal-title">Share Poll</h2>
            <button className="modal-close" onClick={onClose} id="share-modal-close">
              ✕
            </button>
          </div>

          <div className="share-link-group">
            <input
              id="share-link-input"
              className="share-link-input"
              value={shareURL}
              readOnly
            />
            <button className="btn btn-primary btn-sm" onClick={handleCopy} id="copy-link-btn">
              {copied ? '✓' : 'Copy'}
            </button>
          </div>

          <div className="share-qr">
            <img src={qrURL} alt="QR code for poll" loading="lazy" />
          </div>
        </div>
      </div>

      {copied && <div className="copied-toast">Link copied to clipboard!</div>}
    </>
  );
}
