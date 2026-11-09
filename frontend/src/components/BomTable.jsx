// ============================================================
//  TABELA "LISTA DE PEÇAS" (a lista de materiais da máquina)
// ============================================================
//  Mostra as peças agrupadas (ex.: Elétrica > Resistor), com busca,
//  filtros, cores de estoque e o botão "Exportar Excel".
//  Clicar numa linha abre a janela de edição da peça.

import { useMemo, useState } from 'react'
import { money, qty, fornTexto, sigla, outros } from '../format.js'
import { baixarPlanilha, agora } from '../planilha.js'
import Fornecedores from './Fornecedores.jsx'
import SkuHover from './SkuHover.jsx'

const TIPO = { peca: 'Peça', consumivel: 'Consumível', servico: 'Serviço' }

// vermelho = zerado; laranja = abaixo do mínimo
// Cor do estoque: vermelho = zerado; laranja = abaixo do mínimo.
function estoqueClasse(l) {
  if (l.quantidade > 0 && l.peca.estoque <= 0) return 'est-zero'
  if (l.minimo > 0 && l.peca.estoque < l.minimo) return 'est-baixo'
  return ''
}

// "useMemo" guarda um resultado já calculado e só recalcula quando algo
// de que ele depende muda (deixa a tela rápida mesmo com muitas peças).

export default function BomTable({ nomeMaquina = '', linhas, legenda, kitNome, onEdit, soAbaixo = false, onSoAbaixo, onAbrirSub }) {
  const [busca, setBusca] = useState('')
  const [soCompart, setSoCompart] = useState(false) // só peças usadas em mais de uma máquina
  const [grupo, setGrupo] = useState('')
  const [kit, setKit] = useState('todos')

  const grupos = useMemo(() => [...new Set(linhas.map((l) => l.peca.grupo))], [linhas])

  // Aplica os filtros: só abaixo do mínimo, grupo, kit e o texto da busca
  // (procura no SKU, nome, fornecedor, código do fornecedor e uso).
  const filtradas = useMemo(() => {
    const q = busca.trim().toLowerCase()
    return linhas.filter((l) => {
      if (soAbaixo && !(l.minimo > 0 && l.peca.estoque < l.minimo)) return false
      if (soCompart && !(l.usada_em?.length > 1)) return false
      if (grupo && l.peca.grupo !== grupo) return false
      if (kit === 'padrao' && l.kit) return false
      if (kit !== 'todos' && kit !== 'padrao' && l.kit !== kit) return false
      if (!q) return true
      return [l.sku, l.peca.nome, fornTexto(l.peca.fornecedores), l.peca.classe, ...l.onde_usado]
        .join(' ')
        .toLowerCase()
        .includes(q)
    })
  }, [linhas, busca, grupo, kit, soAbaixo, soCompart])

  // agrupa por grupo > classe, mantendo a ordem da planilha
  // Agrupa as peças por "grupo > classe" e soma o custo de cada grupo.
  const secoes = useMemo(() => {
    const out = []
    for (const l of filtradas) {
      const key = `${l.peca.grupo} › ${l.peca.classe}`
      let s = out.find((x) => x.key === key)
      if (!s) out.push((s = { key, grupo: l.peca.grupo, classe: l.peca.classe, linhas: [], custo: 0 }))
      s.linhas.push(l)
      s.custo += l.custo_por_maquina || 0
    }
    return out
  }, [filtradas])

  const kitsExistentes = [...new Set(linhas.map((l) => l.kit).filter(Boolean))]

  // Botão "Exportar Excel": baixa exatamente o que está na tela (com
  // filtros), separado pelas mesmas seções da tela (Elétrica › Resistores...).
  const exportar = () =>
    baixarPlanilha(`lista-pecas-${nomeMaquina.replace('BoxCubo ', '').normalize('NFD').replace(/[\u0300-\u036f]/g, '').replace(/\W+/g, '-').toLowerCase() || 'maquina'}.xlsx`, {
      titulo: `Lista de peças — ${nomeMaquina.replace('BoxCubo ', '')}`,
      subtitulo: `${filtradas.length} itens · gerado em ${agora()}`,
      aba: 'Lista de peças',
      colunas: [
        { titulo: 'SKU' },
        { titulo: 'Peça' },
        { titulo: 'Tipo' },
        { titulo: 'Qtd/máquina', tipo: 'num' },
        { titulo: 'Un.' },
        { titulo: 'Valor unit.', tipo: 'preco' },
        { titulo: 'Custo/máquina', tipo: 'dinheiro', total: true },
        { titulo: 'Estoque', tipo: 'num' },
        { titulo: 'Fornecedor' },
        { titulo: 'Código no fornecedor' },
        { titulo: 'Outros fornecedores' },
        { titulo: 'Usada em' },
        { titulo: 'Onde é usado' },
        { titulo: 'Kit' },
      ],
      linhas: secoes.flatMap((s) =>
        s.linhas.map((l) => ({
          grupo: s.key,
          valores: [
            l.sku, l.peca.nome, l.peca.fabricado ? 'Subconjunto' : TIPO[l.peca.tipo], l.quantidade, l.peca.unidade,
            l.peca.valor_unitario, l.custo_por_maquina, l.peca.estoque,
            l.peca.fornecedores?.[0]?.nome || '', l.peca.fornecedores?.[0]?.codigo || '', outros(l.peca.fornecedores),
            (l.usada_em || []).map(sigla).join(', '), l.onde_usado.join(', '), l.kit ? kitNome(l.kit) : '',
          ],
        })),
      ),
    })

  return (
    <div>
      <div className="toolbar">
        <input
          className="search"
          placeholder="Buscar por SKU, nome, fornecedor, código do fornecedor…"
          value={busca}
          onChange={(e) => setBusca(e.target.value)}
        />
        <select value={grupo} onChange={(e) => setGrupo(e.target.value)}>
          <option value="">Todos os grupos</option>
          {grupos.map((g) => <option key={g}>{g}</option>)}
        </select>
        <select value={kit} onChange={(e) => setKit(e.target.value)}>
          <option value="todos">Padrão + kits</option>
          <option value="padrao">Só itens padrão</option>
          {kitsExistentes.map((k) => <option key={k} value={k}>Só kit {kitNome(k)}</option>)}
        </select>
        {linhas.some((l) => l.minimo > 0) && (
          <label className="check">
            <input type="checkbox" checked={soAbaixo} onChange={(e) => onSoAbaixo?.(e.target.checked)} />
            Só abaixo do mínimo
          </label>
        )}
        <label className="check" title="Peças usadas em mais de uma máquina">
          <input type="checkbox" checked={soCompart} onChange={(e) => setSoCompart(e.target.checked)} />
          Só compartilhadas
        </label>
        <span className="muted">{filtradas.length} itens</span>
        <button className="btn" onClick={exportar}>Exportar Excel</button>
      </div>

      <div className="table-wrap">
        <table className="tbl">
          <thead>
            <tr>
              <th>SKU</th>
              <th>Nome</th>
              <th title="Máquinas que usam esta peça (direto ou dentro de um subconjunto)">Usada em</th>
              <th>Onde é usado</th>
              <th className="r">Qtd/máq.</th>
              <th>Und.</th>
              <th className="r">Valor unit.</th>
              <th className="r">Custo/máq.</th>
              <th>Fornecedor</th>
              <th className="r">Estoque</th>
            </tr>
          </thead>
          <tbody>
            {secoes.map((s) => (
              <Secao key={s.key} s={s} legenda={legenda} kitNome={kitNome} onEdit={onEdit} onAbrirSub={onAbrirSub} />
            ))}
          </tbody>
        </table>
        {secoes.length === 0 && <div className="muted pad">Nenhum item encontrado.</div>}
      </div>
    </div>
  )
}

// Desenha um grupo: a linha de título (ex.: "RESISTOR (12)") e as peças.
function Secao({ s, legenda, kitNome, onEdit, onAbrirSub }) {
  return (
    <>
      <tr className="sec">
        <td colSpan={7}>
          <span className="sec-grupo">{s.grupo}</span> <b>{s.classe}</b> <span className="muted">({s.linhas.length})</span>
        </td>
        <td className="r">{money(s.custo)}</td>
        <td colSpan={2}></td>
      </tr>
      {s.linhas.map((l) => (
        <tr key={l.sku} className="row" onClick={() => onEdit(l)} title="Clique para editar">
          <td><SkuHover sku={l.sku} nome={l.peca.nome} imagem={l.peca.imagem} /></td>
          <td>
            {l.peca.nome}
            {l.kit && <span className="tag kit">{kitNome(l.kit)}</span>}
            {l.peca.tipo !== 'peca' && <span className="tag">{TIPO[l.peca.tipo]}</span>}
            {l.peca.fabricado && (
              <button
                type="button"
                className="tag sub-tag"
                title="Subconjunto fabricado: clique para ver a receita"
                onClick={(e) => {
                  e.stopPropagation()
                  onAbrirSub?.(l.sku)
                }}
              >
                Subconjunto ›
              </button>
            )}
          </td>
          <td>
            {(l.usada_em || []).map((m) => (
              <span key={m} className={`chip maq ${l.usada_em.length > 1 ? 'compart' : ''}`} title={m}>
                {sigla(m)}
              </span>
            ))}
          </td>
          <td>
            {l.onde_usado.map((o) => (
              <span key={o} className="chip" title={legenda[o] || 'Código fora da legenda'}>{o}</span>
            ))}
          </td>
          <td className="r">{qty(l.quantidade)}</td>
          <td>{l.peca.unidade}</td>
          <td className="r">
            {l.peca.valor_unitario == null && l.custo_unitario != null ? (
              <span className="calc" title="Soma da receita do subconjunto">≈ {money(l.custo_unitario)}</span>
            ) : (
              money(l.peca.valor_unitario)
            )}
          </td>
          <td className="r">{money(l.custo_por_maquina)}</td>
          <td className="forn"><Fornecedores lista={l.peca.fornecedores} /></td>
          <td className={`r ${estoqueClasse(l)}`} title={l.minimo > 0 ? `Mínimo: ${qty(l.minimo)} ${l.peca.unidade}` : undefined}>
            {qty(l.peca.estoque)}
            {l.minimo > 0 && <small className="min-hint">mín. {qty(l.minimo)}</small>}
          </td>
        </tr>
      ))}
    </>
  )
}
