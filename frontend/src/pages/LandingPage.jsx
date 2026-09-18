import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export default function LandingPage() {
  const { user } = useAuth();

  return (
    <div className="page-container">
      {/* Hero Section */}
      <section className="landing-hero">
        <div className="hero-badge">
          <span className="hero-badge-dot"></span>
          Real-time audience engagement
        </div>

        <h1 className="hero-title">
          Create Live Polls
          <br />
          <span className="hero-title-gradient">In Seconds</span>
        </h1>

        <p className="hero-subtitle">
          Engage your audience with instant polls. Share a link, collect votes,
          and watch results update live — no refresh needed.
        </p>

        <div className="hero-actions">
          {user ? (
            <>
              <Link to="/create" className="btn btn-primary btn-lg" id="hero-create-btn">
                Create a Poll
              </Link>
              <Link to="/dashboard" className="btn btn-secondary btn-lg" id="hero-dashboard-btn">
                Your Dashboard
              </Link>
            </>
          ) : (
            <>
              <Link to="/signup" className="btn btn-primary btn-lg" id="hero-signup-btn">
                Get Started Free
              </Link>
              <Link to="/login" className="btn btn-secondary btn-lg" id="hero-login-btn">
                Sign In
              </Link>
            </>
          )}
        </div>
      </section>

      {/* Features Section */}
      <section className="features-section">
        <div className="features-grid">
          <div className="glass-card feature-card">
            <div className="feature-icon feature-icon-purple">⚡</div>
            <h3 className="feature-title">Truly Real-Time</h3>
            <p className="feature-desc">
              Votes appear instantly for everyone watching. Powered by
              Server-Sent Events and Redis Pub/Sub — zero polling, zero refresh.
            </p>
          </div>

          <div className="glass-card feature-card">
            <div className="feature-icon feature-icon-cyan">🔗</div>
            <h3 className="feature-title">One Link to Share</h3>
            <p className="feature-desc">
              Each poll gets a unique shareable link and QR code. Perfect for
              meetings, classrooms, and events of any size.
            </p>
          </div>

          <div className="glass-card feature-card">
            <div className="feature-icon feature-icon-orange">🛡️</div>
            <h3 className="feature-title">Fair & Secure</h3>
            <p className="feature-desc">
              Duplicate votes are blocked automatically. Rate limiting prevents
              abuse. Your polls stay honest and reliable.
            </p>
          </div>
        </div>
      </section>
    </div>
  );
}
