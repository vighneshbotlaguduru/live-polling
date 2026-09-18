import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export default function Navbar() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/');
  };

  return (
    <nav className="navbar" id="main-navbar">
      <Link to="/" className="navbar-brand">
        <span className="brand-icon">◉</span>
        <span>LivePoll</span>
      </Link>
      <div className="navbar-links">
        {user ? (
          <>
            <Link to="/dashboard" className="nav-link" id="nav-dashboard">
              Dashboard
            </Link>
            <Link to="/create" className="nav-link nav-link-primary" id="nav-create">
              + Create Poll
            </Link>
            <button
              onClick={handleLogout}
              className="nav-link nav-link-ghost"
              id="nav-logout"
            >
              Logout
            </button>
          </>
        ) : (
          <>
            <Link to="/login" className="nav-link" id="nav-login">
              Log In
            </Link>
            <Link to="/signup" className="nav-link nav-link-primary" id="nav-signup">
              Sign Up
            </Link>
          </>
        )}
      </div>
    </nav>
  );
}
