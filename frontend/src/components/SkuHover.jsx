// ============================================================
//  SKU COM FOTO E BOTÃO DE COPIAR
// ============================================================
//  Usado em todas as tabelas. Mostra o SKU; ao passar o mouse, abre um
//  balão com a foto e o nome da peça; o ícone ao lado copia o código.

import { useState } from 'react'

// Copia um texto para a área de transferência (Ctrl+V cola depois).
// Tem um "plano B" para navegadores que não aceitam o jeito moderno.
async function copiarTexto(txt) {
  try {
    await navigator.clipboard.writeText(txt)
  } catch {
    // reserva para navegadores sem a API de área de transferência
    const ta = document.createElement('textarea')
    ta.value = txt
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    ta.remove()
  }
}

// Mostra o SKU com botão de copiar; ao passar o mouse, abre um balão com a foto.
// Usa position: fixed para não ser cortado pela rolagem das tabelas.
// "pos" = onde desenhar o balão da foto; "copiado" = mostra "Copiado!".
export default function SkuHover({ sku, nome, imagem }) {
  const [pos, setPos] = useState(null)
  const [copiado, setCopiado] = useState(false)

  // Clique no ícone de copiar. O stopPropagation impede que o clique abra
  // também a janela de edição da peça (que é o clique na linha).
  async function copiar(e) {
    e.stopPropagation() // não abre a edição da peça
    await copiarTexto(sku)
    setCopiado(true)
    setTimeout(() => setCopiado(false), 1200)
  }

  // Calcula onde abrir o balão: embaixo do SKU; se não couber na tela,
  // abre para cima.
  function abrir(e) {
    const r = e.currentTarget.getBoundingClientRect()
    const largura = 240
    const altura = 280
    const left = Math.min(r.left, window.innerWidth - largura - 12)
    // abre para baixo; se não couber, para cima
    const top = r.bottom + altura + 12 > window.innerHeight ? r.top - altura - 8 : r.bottom + 8
    setPos({ left, top })
  }

  return (
    <span className="sku-hover" onMouseEnter={abrir} onMouseLeave={() => setPos(null)}>
      <code>{sku}</code>
      {imagem && <span className="has-img" title="Tem foto">●</span>}
      <button
        type="button"
        className={`copiar ${copiado ? 'ok' : ''}`}
        onClick={copiar}
        onMouseEnter={(e) => e.stopPropagation()}
        title="Copiar SKU"
        aria-label={`Copiar ${sku}`}
      >
        {copiado ? (
          'Copiado!'
        ) : (
          <svg viewBox="0 0 16 16" width="13" height="13" fill="none" stroke="currentColor" strokeWidth="1.4">
            <rect x="5" y="5" width="9" height="9" />
            <path d="M3 11V2h9" />
          </svg>
        )}
      </button>
      {pos && (
        <span className="sku-pop" style={{ left: pos.left, top: pos.top }}>
          {imagem ? (
            <img src={imagem} alt={nome || sku} />
          ) : (
            <span className="sku-pop-vazio">
              <svg viewBox="0 0 32 32" width="40" height="40" fill="none" stroke="currentColor" strokeWidth="1.5">
                <path d="M16 3 27.3 9.5v13L16 29 4.7 22.5v-13z" />
              </svg>
              Sem foto
              <small>Clique na peça para adicionar</small>
            </span>
          )}
          <span className="sku-pop-info">
            <code>{sku}</code>
            {nome && <span>{nome}</span>}
          </span>
        </span>
      )}
    </span>
  )
}
