// ============================================================
//  PONTO DE PARTIDA DA TELA
// ============================================================
//  O navegador abre o index.html, que chama este arquivo. Ele "desenha"
//  o componente App (a tela inteira) dentro da página e carrega o visual
//  (styles.css).
//
//  Um "componente" em React é um pedaço da tela que sabe se desenhar
//  sozinho: a barra lateral, a tabela, uma janela... A tela inteira é
//  montada juntando componentes, como peças de LEGO.

import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.jsx'
import './styles.css'

createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
