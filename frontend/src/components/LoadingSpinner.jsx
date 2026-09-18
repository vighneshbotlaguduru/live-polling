export default function LoadingSpinner({ fullPage = false }) {
  return (
    <div className={`spinner-container ${fullPage ? 'spinner-page' : ''}`}>
      <div className="spinner"></div>
    </div>
  );
}
