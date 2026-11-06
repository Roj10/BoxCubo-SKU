// ============================================================
//  ABA "PLANEJAR LOTE"
// ============================================================
//  Você informa quantas máquinas vai fabricar, a margem e se leva o kit.
//  O backend calcula, peça por peça, quanto precisa e quanto falta comprar
//  (a conta está explicada em backend/internal/api/api.go, função plan).

import { useEffect, useState } from 'react'
import { api } from '../api.js'
import { money, qty, downloadCSV, FORN_COLS, fornCols, porFornecedor } from '../format.js'
import Fornecedores from './Fornecedores.jsx'
import EntradaModal from './EntradaModal.jsx'
import SkuHover from './SkuHover.jsx'

// Memórias: quantidade, margem, kits marcados, o plano calculado e se a
// janela de "Dar entrada desta lista" está aberta.
export default function PlanView({ machine, kits, onEstoqueMudou }) {
  const [quantidade, setQuantidade] = useState(43)
  const [margem, setMargem] = useState(10)
  const [kitsSel, setKitsSel] = useState([])
  const [soFaltando, setSoFaltando] = useState(true)
  const [plan, setPlan] = useState(null)
  const [erro, setErro] = useState('')
  const [entrada, setEntrada] = useState(false)
  const [versao, setVersao] = useState(0) // força recalcular depois de uma entrada

  // Sempre que quantidade/margem/kits mudam, pede o cálculo de novo.
  // Espera 250 ms depois da última tecla para não pedir a cada número digitado.
  useEffect(() => {
    const t = setTimeout(() => {
      api
        .plan(machine.id, { quantidade: quantidade || 1, margem: margem || 0, kits: kitsSel })
        .then((p) => {
          setPlan(p)
          setErro('')
        })
        .catch((e) => setErro(e.message))
    }, 250)
    return () => clearTimeout(t)
  }, [machine.id, quantidade, margem, kitsSel, versao])

  const toggleKit = (id) =>
    setKitsSel((ks) => (ks.includes(id) ? ks.filter((k) => k !== id) : [...ks, id]))

  const linhas = plan ? plan.linhas.filter((l) => !soFaltando || l.comprar > 0) : []

  // "Exportar lista de compra": CSV ordenado por fornecedor, com o código do
  // item em cada fornecedor.
  const exportar = () =>
    downloadCSV(`compra-${machine.id}-${quantidade}maq.csv`, [
      [...FORN_COLS, 'SKU', 'Nome', 'Unidade', 'Por máquina', 'Necessário', `Com margem ${margem}%`, 'Estoque', 'Comprar', 'Valor unit.', 'Custo compra'],
      // agrupado por fornecedor principal, para facilitar os pedidos
      ...[...linhas].sort(porFornecedor).map((l) => [
        ...fornCols(l.fornecedores), l.sku, l.nome, l.unidade, l.por_maquina, l.necessario, l.com_margem,
        l.estoque, l.comprar, l.valor_unitario, l.custo_compra,
      ]),
    ])

  return (
    <div>
      <div className="plan-form">
        <label>
          Máquinas no lote
          <input type="number" min="1" value={quantidade} onChange={(e) => setQuantidade(+e.target.value)} />
        </label>
        <label>
          Margem de compra (%)
          <input type="number" min="0" value={margem} onChange={(e) => setMargem(+e.target.value)} />
        </label>
        {kits.map((k) => (
          <label key={k.id} className="check" title={k.descricao}>
            <input type="checkbox" checked={kitsSel.includes(k.id)} onChange={() => toggleKit(k.id)} />
            Incluir kit {k.nome}
          </label>
        ))}
      </div>

      {erro && <div className="alert">{erro}</div>}

      {plan && (
        <>
          <section className="cards">
            <div className="card">
              <span>Custo total do lote</span>
              <strong>{money(plan.custo_lote)}</strong>
            </div>
            <div className="card accent">
              <span>Falta comprar</span>
              <strong>{money(plan.custo_compra)}</strong>
            </div>
            <div className="card">
              <span>Itens a comprar</span>
              <strong>{plan.itens_faltando} de {plan.linhas.length}</strong>
            </div>
            {plan.itens_sem_preco > 0 && (
              <div className="card warn">
                <span>Sem preço (fora do custo)</span>
                <strong>{plan.itens_sem_preco}</strong>
              </div>
            )}
          </section>

          <div className="toolbar">
            <label className="check">
              <input type="checkbox" checked={soFaltando} onChange={(e) => setSoFaltando(e.target.checked)} />
              Mostrar só o que falta comprar
            </label>
            <span className="muted">{linhas.length} itens</span>
            <button className="btn" onClick={exportar}>Exportar lista de compra</button>
            <button
              className="btn primary no-auto"
              disabled={!plan.linhas.some((l) => l.comprar > 0)}
              onClick={() => setEntrada(true)}
              title="Quando o material desta lista chegar"
            >
              Dar entrada desta lista
            </button>
          </div>

          <div className="table-wrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>SKU</th>
                  <th>Nome</th>
                  <th className="r">Por máq.</th>
                  <th className="r">Necessário</th>
                  <th className="r">Com margem</th>
                  <th className="r">Estoque</th>
                  <th className="r">Comprar</th>
                  <th className="r">Custo compra</th>
                  <th>Fornecedor</th>
                </tr>
              </thead>
              <tbody>
                {linhas.map((l) => (
                  <tr key={l.sku}>
                    <td><SkuHover sku={l.sku} nome={l.nome} imagem={l.imagem} /></td>
                    <td>{l.nome}</td>
                    <td className="r">{qty(l.por_maquina)} {l.unidade}</td>
                    <td className="r">{qty(l.necessario)}</td>
                    <td className="r">{qty(l.com_margem)}</td>
                    <td className="r">{qty(l.estoque)}</td>
                    <td className={`r ${l.comprar > 0 ? 'neg' : 'ok'}`}>{qty(l.comprar)}</td>
                    <td className="r">{money(l.custo_compra)}</td>
                    <td className="forn"><Fornecedores lista={l.fornecedores} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {entrada && plan && (
        <EntradaModal
          titulo={`Compra do lote de ${plan.quantidade} ${machine.nome.replace('BoxCubo ', '')}`}
          inicial={plan.linhas.filter((l) => l.comprar > 0).map((l) => ({ sku: l.sku, quantidade: l.comprar }))}
          onClose={() => setEntrada(false)}
          onDone={() => {
            setEntrada(false)
            setVersao((v) => v + 1)
            onEstoqueMudou?.()
          }}
        />
      )}
    </div>
  )
}
