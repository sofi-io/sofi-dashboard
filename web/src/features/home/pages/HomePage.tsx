import { useEffect, useState } from 'react'

import type {
  GeneralStats
} from '../../../types/financial'

export function HomePage() {
    const [stats, setStats] = useState<GeneralStats | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    return (
        <div className="flex flex-col gap-4">
            <h1 className="text-2xl font-bold">Página Inicial</h1>
            <p>Bem-vindo ao painel de controle da Sofi!</p>
        </div>
    )
}