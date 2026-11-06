// ============================================================
//  CAMPO "ONDE É USADO" COM SUGESTÕES
// ============================================================
//  Ao digitar, sugere os códigos da legenda (PR = Placa Raspberry...).
//    - 1 letra ("A")   -> códigos que começam com A (ACA, ADE...);
//    - 2+ letras ("cabo") -> também procura pelo significado (CF, CL...).
//  Cada código escolhido vira uma etiqueta com × para remover.

import { useMemo, useRef, useState } from 'react'

// Campo de códigos "onde é usado" com sugestões da legenda.
// Digite parte do código (A → ACA, ADE…) ou do significado (cabo → CF, CL, CR, CS).
export default function CodigosInput({ value, onChange, legenda }) {
  const [texto, setTexto] = useState('')
  const [aberto, setAberto] = useState(false)
  const [ativo, setAtivo] = useState(0)
  const inputRef = useRef(null)

  // Monta a lista de sugestões na ordem: começa com o texto, contém o texto,
  // significado contém o texto.
  const sugestoes = useMemo(() => {
    const q = texto.trim().toUpperCase()
    const todos = Object.entries(legenda)
      .filter(([cod]) => !value.includes(cod))
      .map(([cod, sig]) => ({ cod, sig }))
    if (!q) return todos.sort((a, b) => a.cod.localeCompare(b.cod))
    const qMin = texto.trim().toLowerCase()
    // primeiro os códigos que começam com o texto, depois os que contêm,
    // depois os que batem pelo significado
    // com uma letra só, mostra apenas os códigos que começam com ela (A → ACA, ADE…)
    const pontua = ({ cod, sig }) =>
      cod.startsWith(q) ? 0 : q.length < 2 ? 9 : cod.includes(q) ? 1 : sig.toLowerCase().includes(qMin) ? 2 : 9
    return todos
      .map((s) => ({ ...s, p: pontua(s) }))
      .filter((s) => s.p < 9)
      .sort((a, b) => a.p - b.p || a.cod.localeCompare(b.cod))
  }, [texto, legenda, value])

  // Adiciona um código como etiqueta (sem repetir).
  function adicionar(cod) {
    cod = cod.trim().toUpperCase()
    if (cod && !value.includes(cod)) onChange([...value, cod])
    setTexto('')
    setAtivo(0)
    inputRef.current?.focus()
  }

  // Teclado: ↑ ↓ escolhem, Enter/vírgula/Tab adicionam, Backspace com o
  // campo vazio apaga a última etiqueta, Esc fecha a lista.
  function onKeyDown(e) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setAberto(true)
      setAtivo((a) => Math.min(a + 1, sugestoes.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setAtivo((a) => Math.max(a - 1, 0))
    } else if (e.key === 'Enter' || e.key === ',' || e.key === 'Tab') {
      if (!texto.trim()) return // Tab/Enter vazio segue o fluxo normal
      e.preventDefault()
      // escolhe a sugestão destacada; se não houver, usa o que foi digitado
      adicionar(aberto && sugestoes[ativo] ? sugestoes[ativo].cod : texto)
    } else if (e.key === 'Backspace' && !texto && value.length) {
      onChange(value.slice(0, -1))
    } else if (e.key === 'Escape') {
      setAberto(false)
    }
  }

  const desconhecido = texto.trim() && !legenda[texto.trim().toUpperCase()]

  return (
    <div className="codigos">
      <div className="codigos-box" onClick={() => inputRef.current?.focus()}>
        {value.map((cod) => (
          <span key={cod} className="codigo-chip" title={legenda[cod] || 'Código fora da legenda'}>
            <b>{cod}</b>
            {legenda[cod] && <span>{legenda[cod]}</span>}
            <button
              type="button"
              aria-label={`Remover ${cod}`}
              onClick={(e) => {
                e.stopPropagation()
                onChange(value.filter((c) => c !== cod))
              }}
            >
              ×
            </button>
          </span>
        ))}
        <input
          ref={inputRef}
          value={texto}
          onChange={(e) => {
            setTexto(e.target.value)
            setAberto(true)
            setAtivo(0)
          }}
          onFocus={() => setAberto(true)}
          onBlur={() => setTimeout(() => setAberto(false), 150)}
          onKeyDown={onKeyDown}
          placeholder={value.length ? '' : 'Digite um código ou o que é (ex.: A, cabo, placa)'}
        />
      </div>

      {aberto && (sugestoes.length > 0 || desconhecido) && (
        <ul className="codigos-lista" role="listbox">
          {sugestoes.map((s, i) => (
            <li
              key={s.cod}
              role="option"
              aria-selected={i === ativo}
              className={i === ativo ? 'on' : ''}
              onMouseEnter={() => setAtivo(i)}
              onMouseDown={(e) => {
                e.preventDefault()
                adicionar(s.cod)
              }}
            >
              <b>{s.cod}</b>
              <span>{s.sig}</span>
            </li>
          ))}
          {desconhecido && sugestoes.length === 0 && (
            <li className="novo" onMouseDown={(e) => {
              e.preventDefault()
              adicionar(texto)
            }}>
              Usar <b>{texto.trim().toUpperCase()}</b> (fora da legenda)
            </li>
          )}
        </ul>
      )}
    </div>
  )
}
