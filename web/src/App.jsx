
import React, { Suspense } from 'react';
import { Navigate, Route, Routes, useLocation } from 'react-router-dom';
import Loading from './components/common/ui/Loading';
import { AuthRedirect, PrivateRoute } from './helpers';
import LoginForm from './components/auth/LoginForm';
import NotFound from './pages/NotFound';
import Setting from './pages/Setting';
import Channel from './pages/Channel';
import Token from './pages/Token';
import ModelPage from './pages/Model';
import PersonalSetting from './components/settings/PersonalSetting';
import Setup from './pages/Setup';
import SetupCheck from './components/layout/SetupCheck';

import { TooltipProvider } from '@/components/ui/tooltip';

function App() {
  const location = useLocation();

  return (
    <TooltipProvider>
      <SetupCheck>
      <Routes>
        <Route
          path='/'
          element={
            <Suspense fallback={<Loading />} key={location.pathname}>
              <AuthRedirect>
                <LoginForm />
              </AuthRedirect>
            </Suspense>
          }
        />
        <Route
          path='/login'
          element={
            <Suspense fallback={<Loading />} key={location.pathname}>
              <AuthRedirect>
                <LoginForm />
              </AuthRedirect>
            </Suspense>
          }
        />
        <Route
          path='/setup'
          element={
            <Suspense fallback={<Loading />} key={location.pathname}>
              <Setup />
            </Suspense>
          }
        />
        <Route
          path='/console'
          element={
            <PrivateRoute>
              <Navigate to='/console/channel' replace />
            </PrivateRoute>
          }
        />
        <Route
          path='/console/channel'
          element={
            <PrivateRoute>
              <Channel />
            </PrivateRoute>
          }
        />
        <Route
          path='/console/models'
          element={
            <PrivateRoute>
              <ModelPage />
            </PrivateRoute>
          }
        />
        <Route
          path='/console/token'
          element={
            <PrivateRoute>
              <Token />
            </PrivateRoute>
          }
        />
        <Route
          path='/console/setting'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading />} key={location.pathname}>
                <Setting />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route
          path='/console/personal'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading />} key={location.pathname}>
                <PersonalSetting />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route path='*' element={<NotFound />} />
      </Routes>
    </SetupCheck>
    </TooltipProvider>
  );
}

export default App;
