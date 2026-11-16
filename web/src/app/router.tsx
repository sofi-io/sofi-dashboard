
import { createBrowserRouter } from 'react-router-dom'

import { AppLayout } from '../components/layout/AppLayout'
import { CompetitionsPage } from '../features/competitions/pages/CompetitionsPage'
import { IntelligencePage } from '../features/intelligence/pages/IntelligencePage'
import { CompetitionDetailPage } from '../features/competitions/pages/CompetitionDetailPage'

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
        element: <CompetitionDetailPage />,
      },
      {
        path: '/inteligencia',
        element: <IntelligencePage />,
      },
    ],
  },
])
