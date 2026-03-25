
import React from 'react';
import { Navigate } from 'react-router-dom';
import { history } from './history';

export function authHeader() {
  const user = JSON.parse(localStorage.getItem('user'));

  if (user && user.token) {
    return { Authorization: 'Bearer ' + user.token };
  }
  return {};
}

export const AuthRedirect = ({ children }) => {
  const user = localStorage.getItem('user');

  if (user) {
    return <Navigate to='/console' replace />;
  }

  return children;
};

function PrivateRoute({ children }) {
  if (!localStorage.getItem('user')) {
    return <Navigate to='/login' state={{ from: history.location }} />;
  }
  return children;
}

export { PrivateRoute };
