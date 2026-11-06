// ============================================================
//  BARRA LATERAL (a coluna escura à esquerda)
// ============================================================
//  Mostra o logo, a lista de máquinas (01 Estática V2, 02 V2.5...) e os
//  atalhos de estoque (Movimentações e Reposição, com o número laranja de
//  peças abaixo do mínimo).

import { useEffect, useState } from 'react'
import { api } from '../api.js'

// Logo: hexágono com borda laranja + nome.
// Logo: hexágono laranja + "BOXCUBO".
function Logo() {
  return (
    <div className="logo">
      <svg viewBox="0 0 32 32" width="30" height="30" fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round">
        <path d="M16 3 27.3 9.5v13L16 29 4.7 22.5v-13z" />
      </svg>
      <span>BOXCUBO</span>
    </div>
  )
}

// "machines" = lista de máquinas; "selected" = o que está aberto agora;
// "onSelect" = o que fazer quando alguém clica num item.
export default function Sidebar({ machines, selected, onSelect }) {
  const [abaixo, setAbaixo] = useState(0)

  // contador de peças abaixo do mínimo, atualizado sempre que o estoque muda
  // Busca quantas peças estão abaixo do mínimo e fica "ouvindo" o aviso de
  // estoque mudou para atualizar o número sozinho.
  useEffect(() => {
    const carregar = () => api.reposicao().then((r) => setAbaixo(r.itens)).catch(() => {})
    carregar()
    window.addEventListener('estoque-mudou', carregar)
    return () => window.removeEventListener('estoque-mudou', carregar)
  }, [])

  return (
    <aside className="sidebar">
      <div className="brand">
        <Logo />
        <span className="brand-sub">Sistema de SKU</span>
      </div>

      <div className="side-title">Máquinas</div>
      <nav>
        {machines.map((m, i) => (
          <button
            key={m.id}
            className={`side-item ${selected === m.id ? 'active' : ''} ${m.ativa ? '' : 'off'}`}
            onClick={() => onSelect(m.id)}
          >
            <span className="num">{String(i + 1).padStart(2, '0')}</span>
            <span className="side-body">
              <span className="side-name">{m.nome.replace('BoxCubo ', '')}</span>
              <span className="side-apelido">{m.apelido}</span>
              <span className="side-meta">
                <code>{m.sku}</code>
                {!m.ativa && <em className="soon">Em breve</em>}
              </span>
            </span>
          </button>
        ))}
      </nav>

      <div className="side-title">Estoque</div>
      <nav>
        <button
          className={`side-item ${selected === 'mov' ? 'active' : ''}`}
          onClick={() => onSelect('mov')}
        >
          <span className="num ghost">⇅</span>
          <span className="side-body">
            <span className="side-name">Movimentações</span>
            <span className="side-apelido">Entradas, saídas e estornos</span>
          </span>
        </button>
        <button
          className={`side-item ${selected === 'repo' ? 'active' : ''}`}
          onClick={() => onSelect('repo')}
        >
          <span className="num ghost">!</span>
          <span className="side-body">
            <span className="side-name">
              Reposição
              {abaixo > 0 && <span className="badge" title="Peças abaixo do estoque mínimo">{abaixo}</span>}
            </span>
            <span className="side-apelido">Abaixo do estoque mínimo</span>
          </span>
        </button>
      </nav>

      <div className="side-foot">
        <a href="https://boxcubo.com.br/" target="_blank" rel="noreferrer">boxcubo.com.br ↗</a>
      </div>
    </aside>
  )
}
