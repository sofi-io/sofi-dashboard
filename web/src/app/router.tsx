
import { createBrowserRouter } from 'react-router-dom'

import { AppLayout } from '../components/layout/AppLayout'
import { CompetitionsPage } from '../features/competitions/pages/CompetitionsPage'
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
        path: '/inteligencia',
        element: <IntelligencePage />,
      },
    ],
  },
])
