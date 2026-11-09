// ============================================================
//  TELA "MOVIMENTAÇÕES" (o histórico do estoque)
// ============================================================
//  Lista tudo que entrou e saiu, do mais novo para o mais antigo.
//  Clicar numa linha abre o detalhe peça por peça (antes, depois, falta).
//  "Estornar" desfaz uma movimentação lançada errada.

import { useCallback, useEffect, useState } from 'react'
import { api, avisarEstoque } from '../api.js'
import { qty } from '../format.js'
import EntradaModal from './EntradaModal.jsx'
import SkuHover from './SkuHover.jsx'

const TIPOS = {
  saida_maquina: 'Saída de máquina',
  saida_pecas: 'Saída de peças',
  fabricacao: 'Fabricação',
  ajuste: 'Ajuste de estoque',
  entrada: 'Entrada',
  estorno: 'Estorno',
}

// Data/hora no formato brasileiro: 06/11/2026, 12:16.
const quando = (iso) =>
  new Date(iso).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })

// Memórias: lista carregada, filtros, linha aberta, janela aberta e as
// fotos das peças (para o balão do SKU no detalhe).
export default function Movimentacoes({ machines }) {
  const [movs, setMovs] = useState(null)
  const [tipo, setTipo] = useState('')
  const [busca, setBusca] = useState('')
  const [aberto, setAberto] = useState(null)
  const [modal, setModal] = useState(null) // 'entrada' | 'saida'
  const [erro, setErro] = useState('')
  const [fotos, setFotos] = useState({})

  useEffect(() => {
    api.parts().then((ps) => setFotos(Object.fromEntries(ps.map((p) => [p.sku, p.imagem]))))
  }, [])

  const load = useCallback(() => {
    api.movements({ tipo, q: busca }).then(setMovs).catch((e) => setErro(e.message))
  }, [tipo, busca])

  useEffect(() => {
    const t = setTimeout(load, 200)
    return () => clearTimeout(t)
  }, [load])

  const [subs, setSubs] = useState([]) // subconjuntos (para mostrar "CPU V2" e não o SKU)
  useEffect(() => {
    api.subconjuntos().then(setSubs).catch(() => {})
  }, [])
  const nomeMaquina = (id) =>
    machines.find((m) => m.id === id)?.nome.replace('BoxCubo ', '') ||
    subs.find((s) => s.id === id)?.nome.replace(/\s*\(.*\)\s*$/, '') ||
    id

  // Pergunta o motivo e pede o estorno ao backend.
  async function estornar(m) {
    const motivo = window.prompt(
      `Estornar ${m.id}? O estoque volta ao que era antes desta movimentação.\n\nMotivo (opcional):`,
    )
    if (motivo === null) return
    try {
      await api.estorno(m.id, motivo)
      load()
      avisarEstoque()
    } catch (e) {
      setErro(e.message)
    }
  }

  return (
    <div>
      <header className="page-head">
        <div>
          <div className="eyebrow">Estoque</div>
          <h1>Movimentações</h1>
          <span className="muted">Histórico de tudo que entrou e saiu do estoque.</span>
        </div>
        <div className="head-actions">
          <button className="btn" onClick={() => setModal('saida')}>− Registrar saída de peças</button>
          <button className="btn primary" onClick={() => setModal('entrada')}>+ Registrar entrada</button>
        </div>
      </header>

      <div className="toolbar">
        <input
          className="search"
          placeholder="Buscar por número de série, cliente, MOV-…"
          value={busca}
          onChange={(e) => setBusca(e.target.value)}
        />
        <select value={tipo} onChange={(e) => setTipo(e.target.value)}>
          <option value="">Todos os tipos</option>
          {Object.entries(TIPOS).map(([v, t]) => <option key={v} value={v}>{t}</option>)}
        </select>
        {movs && <span className="muted">{movs.length} movimentações</span>}
      </div>

      {erro && <div className="alert">{erro}</div>}

      <div className="table-wrap">
        <table className="tbl">
          <thead>
            <tr>
              <th>Código</th>
              <th>Data</th>
              <th>Tipo</th>
              <th>Máquina / séries</th>
              <th>Cliente / obs.</th>
              <th className="r">Peças</th>
              <th>Situação</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {movs?.map((m) => (
              <MovRow
                key={m.id}
                m={m}
                aberto={aberto === m.id}
                onToggle={() => setAberto(aberto === m.id ? null : m.id)}
                onEstornar={() => estornar(m)}
                nomeMaquina={nomeMaquina}
                fotos={fotos}
              />
            ))}
          </tbody>
        </table>
        {movs && movs.length === 0 && (
          <div className="muted pad">
            Nenhuma movimentação ainda. Use “Registrar saída” na página da máquina, ou “Registrar saída de peças” / “Registrar entrada” aqui.
          </div>
        )}
      </div>

      {modal && (
        <EntradaModal
          key={modal}
          modo={modal}
          onClose={() => setModal(null)}
          onDone={() => {
            setModal(null)
            load()
          }}
        />
      )}
    </div>
  )
}

// Uma linha do histórico + (se aberta) a tabela de detalhe.
function MovRow({ m, aberto, onToggle, onEstornar, nomeMaquina, fotos }) {
  const faltas = m.itens.filter((i) => i.falta > 0)
  return (
    <>
      <tr className={`row ${m.estornado ? 'riscado' : ''}`} onClick={onToggle}>
        <td><code>{m.id}</code></td>
        <td>{quando(m.data)}</td>
        <td><span className={`tag tipo-${m.tipo}`}>{TIPOS[m.tipo] || m.tipo}</span></td>
        <td>
          {m.machine_id && <b>{m.quantidade ? `${m.quantidade}× ` : ''}{nomeMaquina(m.machine_id)}</b>}
          {m.tipo === 'ajuste' && <span className="muted small">Correção de estoque</span>}
          {m.kits?.length > 0 && <span className="tag kit">+ {m.kits.join(', ')}</span>}
          {m.tipo === 'saida_pecas' && <span className="muted small">Peças avulsas</span>}
          {m.numeros_serie?.length > 0 && (
            <div className="series">
              {m.tipo === 'saida_pecas' && 'Máquina do cliente: '}
              {m.numeros_serie.join(' · ')}
            </div>
          )}
        </td>
        <td className="forn">
          {m.cliente && <div>{m.cliente}</div>}
          {m.estorno_de && <div>Estorno de <code>{m.estorno_de}</code></div>}
          {m.observacao && <div>{m.observacao}</div>}
        </td>
        <td className="r">{m.itens.length}</td>
        <td>
          {m.estornado && <span className="tag">Estornado por {m.estornado_por}</span>}
          {!m.estornado && m.com_falta && <span className="tag falta">{faltas.length} com falta</span>}
          {!m.estornado && !m.com_falta && <span className="ok small">OK</span>}
        </td>
        <td className="r" onClick={(e) => e.stopPropagation()}>
          {m.tipo !== 'estorno' && !m.estornado && (
            <button className="btn tiny" onClick={onEstornar}>Estornar</button>
          )}
        </td>
      </tr>
      {aberto && (
        <tr className="detalhe">
          <td colSpan={8}>
            <table className="tbl inner">
              <thead>
                <tr>
                  <th>SKU</th>
                  <th>Peça</th>
                  <th className="r">Solicitado</th>
                  <th className="r">Movimentado</th>
                  <th className="r">Antes</th>
                  <th className="r">Depois</th>
                  <th className="r">Falta</th>
                </tr>
              </thead>
              <tbody>
                {m.itens.map((i) => (
                  <tr key={i.sku}>
                    <td><SkuHover sku={i.sku} nome={i.nome} imagem={fotos[i.sku]} /></td>
                    <td>{i.nome}</td>
                    <td className="r">{qty(i.solicitado)} {i.unidade}</td>
                    <td className={`r ${i.movimentado < 0 ? '' : 'ok'}`}>
                      {i.movimentado > 0 ? '+' : ''}{qty(i.movimentado)}
                    </td>
                    <td className="r">{qty(i.estoque_antes)}</td>
                    <td className="r">{qty(i.estoque_depois)}</td>
                    <td className={`r ${i.falta > 0 ? 'neg' : 'muted'}`}>{i.falta > 0 ? qty(i.falta) : '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </td>
        </tr>
      )}
    </>
  )
}
