// ============================================================
//  APP: a tela inteira (barra lateral + área principal)
// ============================================================
//  Decide O QUE mostrar na área principal, conforme o que foi clicado na
//  barra lateral:
//    - uma máquina ativa      -> página da máquina (MachinePage)
//    - uma máquina "em breve" -> aviso de que a lista ainda não existe
//    - "Movimentações"        -> histórico (Movimentacoes)
//    - "Reposição"            -> peças abaixo do mínimo (Reposicao)
//    - "Fornecedores"         -> quem fornece o quê (FornecedoresPage)

import { useEffect, useState } from 'react'
import { api } from './api.js'
import Sidebar from './components/Sidebar.jsx'
import MachinePage from './components/MachinePage.jsx'
import Movimentacoes from './components/Movimentacoes.jsx'
import Reposicao from './components/Reposicao.jsx'
import FornecedoresPage from './components/FornecedoresPage.jsx'

// "useState" é a memória da tela: guarda valores que, quando mudam, fazem
// a tela se redesenhar sozinha (ex.: qual item da barra lateral está
// selecionado).
export default function App() {
  const [machines, setMachines] = useState([])
  const [subs, setSubs] = useState([]) // subconjuntos (CPU, Shield...) — abrem na mesma tela de máquina
  const [selected, setSelected] = useState('v2')
  const [erro, setErro] = useState('')

  // "useEffect" roda uma ação quando a tela abre. Aqui: busca a lista de
  // máquinas no backend. Se o backend estiver desligado, mostra o aviso.
  useEffect(() => {
    api.machines().then(setMachines).catch((e) =>
      setErro(`Não consegui falar com o backend (${e.message}). Ele está rodando em localhost:8080?`),
    )
    api.subconjuntos().then(setSubs).catch(() => {})
  }, [])

  const machine = machines.find((m) => m.id === selected) || subs.find((s) => s.id === selected)

  return (
    <div className="layout">
      <Sidebar machines={machines} subs={subs} selected={selected} onSelect={setSelected} />
      <main className="main">
        {erro && <div className="alert">{erro}</div>}
        {selected === 'mov' && <Movimentacoes machines={machines} />}
        {selected === 'repo' && <Reposicao onIrParaMaquina={setSelected} />}
        {selected === 'forn' && <FornecedoresPage />}
        {machine && machine.ativa && (
          <MachinePage
            key={machine.id}
            machine={machine}
            onIrPara={setSelected}
            onMachineUpdated={(m) => setMachines((ms) => ms.map((x) => (x.id === m.id ? m : x)))}
          />
        )}
        {machine && !machine.ativa && (
          <div className="empty">
            <h2>{machine.nome}</h2>
            <p>A lista de peças desta máquina ainda não foi cadastrada.</p>
            <p className="muted">SKU reservado: <code>{machine.sku}</code></p>
          </div>
        )}
      </main>
    </div>
  )
}
