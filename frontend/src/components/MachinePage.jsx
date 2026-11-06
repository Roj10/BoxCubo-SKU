// ============================================================
//  PÁGINA DA MÁQUINA (ex.: Estática V2)
// ============================================================
//  De cima para baixo:
//    1) cabeçalho: nome, SKU e botões "+ Nova peça" e "Registrar saída";
//    2) cartões com números: itens, custo por máquina, kit, sem preço;
//    3) faixa do estoque mínimo;
//    4) abas "Lista de peças" e "Planejar lote";
//    5) as janelas que abrem por cima (editar peça, saída, mínimo).

import { useCallback, useEffect, useState } from 'react'
import { api, avisarEstoque } from '../api.js'
import { money } from '../format.js'
import BomTable from './BomTable.jsx'
import PlanView from './PlanView.jsx'
import PartForm from './PartForm.jsx'
import SaidaModal from './SaidaModal.jsx'
import MinimoModal from './MinimoModal.jsx'

// Memórias desta página: a lista carregada (bom), qual aba está aberta,
// qual janela está aberta (editar/saída/mínimo) e filtros.
export default function MachinePage({ machine, onMachineUpdated, onIrPara }) {
  const [bom, setBom] = useState(null)
  const [kits, setKits] = useState([])
  const [legenda, setLegenda] = useState({})
  const [classes, setClasses] = useState([])
  const [tab, setTab] = useState('pecas')
  const [editing, setEditing] = useState(null) // null | 'new' | BOMLine
  const [erro, setErro] = useState('')
  const [saida, setSaida] = useState(false)
  const [minimo, setMinimo] = useState(false)
  const [soAbaixo, setSoAbaixo] = useState(false)

  // Busca (ou busca de novo) a lista de peças da máquina no backend.
  const load = useCallback(() => {
    api.bom(machine.id).then(setBom).catch((e) => setErro(e.message))
    api.classes().then(setClasses)
  }, [machine.id])

  // Ao abrir a página: carrega a lista, os kits e a legenda (PR, PS...) e
  // fica ouvindo o aviso de estoque mudou para recarregar sozinha.
  useEffect(() => {
    load()
    api.kits().then(setKits)
    api.legenda().then(setLegenda)
    window.addEventListener('estoque-mudou', load)
    return () => window.removeEventListener('estoque-mudou', load)
  }, [load])

  if (!bom) return <div className="muted pad">{erro || 'Carregando…'}</div>

  const kitNome = (id) => kits.find((k) => k.id === id)?.nome || id
  const totalKits = Object.values(bom.custo_por_kit).reduce((a, b) => a + b, 0)
  const temKits = bom.linhas.some((l) => l.kit)
  const nMin = machine.minimo_maquinas || 0

  return (
    <div>
      <header className="page-head">
        <div>
          <div className="eyebrow">{machine.apelido} · {machine.tecnologia}</div>
          <h1>{machine.nome}</h1>
          <code className="sku-big">{machine.sku}</code>
        </div>
        <div className="head-actions">
          <button className="btn" onClick={() => setEditing('new')}>+ Nova peça</button>
          <button className="btn primary" onClick={() => setSaida(true)}>Registrar saída</button>
        </div>
      </header>

      <section className="cards">
        <div className="card">
          <span>Itens na lista</span>
          <strong>{bom.linhas.length}</strong>
        </div>
        <div className="card">
          <span>Custo por máquina (padrão)</span>
          <strong>{money(bom.custo_padrao)}</strong>
        </div>
        {Object.entries(bom.custo_por_kit).map(([k, v]) => (
          <div className="card" key={k}>
            <span>Kit {kitNome(k)}</span>
            <strong>+ {money(v)}</strong>
          </div>
        ))}
        <div className="card">
          <span>Custo com todos os kits</span>
          <strong>{money(bom.custo_padrao + totalKits)}</strong>
        </div>
        {bom.itens_sem_preco > 0 && (
          <div className="card warn">
            <span>Sem preço cadastrado</span>
            <strong>{bom.itens_sem_preco}</strong>
          </div>
        )}
      </section>

      {/* faixa do estoque mínimo */}
      <section className={`minimo-bar ${nMin && bom.abaixo_minimo ? 'alerta' : ''}`}>
        <div className="minimo-info">
          <span className="eyebrow-sm">Estoque mínimo</span>
          {nMin ? (
            <b>
              Peças para {nMin} {nMin === 1 ? 'máquina' : 'máquinas'}
              {machine.minimo_com_kits ? ' (com kits)' : ''}
            </b>
          ) : (
            <b className="muted">Não configurado</b>
          )}
          <button className="btn tiny" onClick={() => setMinimo(true)}>{nMin ? 'Alterar' : 'Configurar'}</button>
        </div>
        {nMin > 0 && (
          <div className="minimo-status">
            {bom.abaixo_minimo > 0 ? (
              <>
                <span className="minimo-num">{bom.abaixo_minimo}</span>
                <span>{bom.abaixo_minimo === 1 ? 'peça abaixo do mínimo' : 'peças abaixo do mínimo'}</span>
                <button
                  className={`btn tiny ${soAbaixo ? 'on' : ''}`}
                  onClick={() => {
                    setTab('pecas')
                    setSoAbaixo(!soAbaixo)
                  }}
                >
                  {soAbaixo ? 'Mostrar todas' : 'Ver na lista'}
                </button>
                <button className="btn tiny" onClick={() => onIrPara('repo')}>Reposição →</button>
              </>
            ) : (
              <span className="ok">✓ Todas as peças acima do mínimo</span>
            )}
          </div>
        )}
      </section>

      <div className="tabs">
        <button className={tab === 'pecas' ? 'on' : ''} onClick={() => setTab('pecas')}>Lista de peças</button>
        <button className={tab === 'plano' ? 'on' : ''} onClick={() => setTab('plano')}>Planejar lote</button>
      </div>

      {tab === 'pecas' && (
        <BomTable
          linhas={bom.linhas}
          legenda={legenda}
          kitNome={kitNome}
          onEdit={setEditing}
          soAbaixo={soAbaixo}
          onSoAbaixo={setSoAbaixo}
        />
      )}
      {tab === 'plano' && <PlanView machine={machine} kits={kits} onEstoqueMudou={avisarEstoque} />}

      {saida && (
        <SaidaModal
          machine={machine}
          kits={kits}
          onClose={() => setSaida(false)}
          onDone={() => {
            setSaida(false)
            avisarEstoque()
          }}
          onIrParaReposicao={() => {
            setSaida(false)
            avisarEstoque()
            onIrPara('repo')
          }}
        />
      )}

      {minimo && (
        <MinimoModal
          machine={machine}
          temKits={temKits}
          onClose={() => setMinimo(false)}
          onSaved={(m) => {
            setMinimo(false)
            onMachineUpdated(m)
          }}
        />
      )}

      {editing && (
        <PartForm
          machine={machine}
          line={editing === 'new' ? null : editing}
          classes={classes}
          kits={kits}
          legenda={legenda}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null)
            avisarEstoque()
          }}
        />
      )}
    </div>
  )
}
