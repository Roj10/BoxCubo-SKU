// ============================================================
//  TELA "REPOSIÇÃO" (peças abaixo do estoque mínimo)
// ============================================================
//  Mostra o que comprar para voltar ao mínimo, com custo e fornecedor.
//  "Exportar Excel" baixa a lista separada por fornecedor, com subtotais; "Dar entrada desta lista"
//  abre a janela de entrada já preenchida, para quando o material chegar.

import { useCallback, useEffect, useState } from 'react'
import { api, avisarEstoque } from '../api.js'
import { money, qty, porFornecedor, grupoFornecedor, outros } from '../format.js'
import { baixarPlanilha, agora } from '../planilha.js'
import Fornecedores from './Fornecedores.jsx'
import SkuHover from './SkuHover.jsx'
import EntradaModal from './EntradaModal.jsx'

// Peças abaixo do estoque mínimo, com sugestão de compra.
// Busca a reposição ao abrir e sempre que o estoque mudar.
export default function Reposicao({ onIrParaMaquina }) {
  const [rep, setRep] = useState(null)
  const [erro, setErro] = useState('')
  const [entrada, setEntrada] = useState(false)

  const load = useCallback(() => {
    api.reposicao().then(setRep).catch((e) => setErro(e.message))
  }, [])

  useEffect(() => {
    load()
    window.addEventListener('estoque-mudou', load)
    return () => window.removeEventListener('estoque-mudou', load)
  }, [load])

  // Planilha do Excel separada por fornecedor principal (um bloco por
  // fornecedor, com subtotal), para montar os pedidos.
  const exportar = () =>
    baixarPlanilha('reposicao.xlsx', {
      titulo: 'Reposição de estoque mínimo',
      subtitulo:
        `${rep.itens} peças abaixo do mínimo · mínimo: ` +
        rep.maquinas.map((m) => `${m.nome.replace('BoxCubo ', '')} ${m.minimo_maquinas} máq.`).join(', ') +
        ` · gerado em ${agora()}`,
      aba: 'Reposição',
      colunas: [
        { titulo: 'SKU' },
        { titulo: 'Peça' },
        { titulo: 'Código no fornecedor' },
        { titulo: 'Un.' },
        { titulo: 'Estoque', tipo: 'num' },
        { titulo: 'Mínimo', tipo: 'num' },
        { titulo: 'Comprar', tipo: 'num', destaque: true },
        { titulo: 'Valor unit.', tipo: 'preco' },
        { titulo: 'Custo', tipo: 'dinheiro', total: true },
        { titulo: 'Outros fornecedores' },
      ],
      linhas: [...rep.linhas].sort(porFornecedor).map((l) => ({
        grupo: grupoFornecedor(l.fornecedores),
        valores: [
          l.sku, l.nome, l.fornecedores?.[0]?.codigo || '', l.unidade, l.estoque, l.minimo, l.sugestao,
          l.valor_unitario, l.custo, outros(l.fornecedores),
        ],
      })),
    })

  return (
    <div>
      <header className="page-head">
        <div>
          <div className="eyebrow">Estoque</div>
          <h1>Reposição</h1>
          <span className="muted">Peças abaixo do estoque mínimo e quanto comprar para voltar a ele.</span>
        </div>
        {rep && rep.itens > 0 && (
          <div className="head-actions">
            <button className="btn" onClick={exportar}>Exportar Excel</button>
            <button className="btn primary" onClick={() => setEntrada(true)}>Dar entrada desta lista</button>
          </div>
        )}
      </header>

      {erro && <div className="alert">{erro}</div>}

      {rep && rep.maquinas.length === 0 && (
        <div className="empty small-empty">
          <h2>Nenhum estoque mínimo configurado</h2>
          <p>
            Na página da máquina, use o cartão <b>Estoque mínimo</b> para dizer para quantas máquinas vocês querem
            ter peças sempre em estoque.
          </p>
          <button className="btn primary" onClick={() => onIrParaMaquina('v2')}>Configurar na Estática V2</button>
        </div>
      )}

      {rep && rep.maquinas.length > 0 && (
        <>
          <section className="cards">
            <div className={`card ${rep.itens ? 'warn' : ''}`}>
              <span>Peças abaixo do mínimo</span>
              <strong>{rep.itens}</strong>
            </div>
            <div className={`card ${rep.zerados ? 'danger' : ''}`}>
              <span>Zeradas</span>
              <strong>{rep.zerados}</strong>
            </div>
            <div className="card accent">
              <span>Custo para repor</span>
              <strong>{money(rep.custo_total)}</strong>
            </div>
            <div className="card">
              <span>Mínimo configurado</span>
              <strong className="card-txt">
                {rep.maquinas.map((m) => (
                  <button key={m.id} className="link" onClick={() => onIrParaMaquina(m.id)}>
                    {m.nome.replace('BoxCubo ', '')}: {m.minimo_maquinas} máq.{m.minimo_com_kits ? ' + kits' : ''}
                  </button>
                ))}
              </strong>
            </div>
          </section>

          {rep.itens === 0 ? (
            <div className="empty small-empty ok-empty">
              <h2>Tudo acima do mínimo</h2>
              <p>Nenhuma peça precisa de reposição agora.</p>
            </div>
          ) : (
            <div className="table-wrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>SKU</th>
                    <th>Peça</th>
                    <th className="r">Estoque</th>
                    <th className="r">Mínimo</th>
                    <th className="r">Comprar</th>
                    <th className="r">Valor unit.</th>
                    <th className="r">Custo</th>
                    <th>Fornecedor</th>
                  </tr>
                </thead>
                <tbody>
                  {rep.linhas.map((l) => (
                    <tr key={l.sku}>
                      <td><SkuHover sku={l.sku} nome={l.nome} imagem={l.imagem} /></td>
                      <td>
                        {l.nome}
                        {l.zerado && <span className="tag zerado">Zerada</span>}
                      </td>
                      <td className={`r ${l.zerado ? 'est-zero' : 'est-baixo'}`}>{qty(l.estoque)} {l.unidade}</td>
                      <td className="r">{qty(l.minimo)}</td>
                      <td className="r neg">{qty(l.sugestao)}</td>
                      <td className="r">{money(l.valor_unitario)}</td>
                      <td className="r">{money(l.custo)}</td>
                      <td className="forn"><Fornecedores lista={l.fornecedores} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {entrada && rep && (
        <EntradaModal
          titulo="Reposição de estoque mínimo"
          inicial={rep.linhas.map((l) => ({ sku: l.sku, quantidade: l.sugestao }))}
          onClose={() => setEntrada(false)}
          onDone={() => {
            setEntrada(false)
            avisarEstoque()
          }}
        />
      )}
    </div>
  )
}
