
import { createBrowserRouter } from 'react-router-dom'

import { AppLayout } from '../components/layout/AppLayout'
import { CompetitionsPage } from '../features/competitions/pages/CompetitionsPage'
import { CompetitionPage } from '../features/competitions/pages/CompetitionPage'
import { IntelligencePage } from '../features/intelligence/pages/IntelligencePage'

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      {
        path: '/',
        element: <CompetitionsPage />,
      },
      {
        path: '/competitions/:id',
        element: <CompetitionPage />,
      },
      {
        path: '/inteligencia',
        element: <IntelligencePage />,
      },
    ],
  },
])
