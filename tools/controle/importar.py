"""Monta backend/internal/store/receitas_controle.json a partir do que foi lido
no sistema Controle BoxCubo (http://192.168.15.10/materiais) em 09/11/2026.

Entradas (nesta pasta):
  materiais.json  -> [nome, unidade, estoque, estoque_cheio, preço, tem_link, fabricado]
  receitas.txt    -> receitas da V2, EYE, Vision e dos subconjuntos CPU e Shield

Cada material do Controle é ligado a um SKU:
  - se a peça já existia no sistema SKU, reaproveita o SKU (MAPA abaixo);
  - se não existia, cria um SKU novo (NOVOS abaixo).
O backend aplica isso uma vez nos dados (migração "receitas-controle-v1").

Uso:  python tools/controle/importar.py
"""
import json
import pathlib
import re

AQUI = pathlib.Path(__file__).resolve().parent
ROOT = AQUI.parent.parent


def norm(s):
    return re.sub(r"\s+", " ", s).strip()


# material do Controle -> SKU que JÁ existe no sistema SKU
MAPA = {
    "Adesivo Modo Cubagem / Modo Balança": "EST-ADE-MODO",
    "Adesivo Objetos Sobre a Mesa": "EST-ADE-NAOAPOIE",
    "Arruela Lisa ZB 3/8 - IMP (Fixa Estrutura)": "EST-PAR-ARR38",
    "Arruela Lisa ZB 3/8 Abalarga (Fixa Balança)": "EST-PAR-ARR38AL",
    "Balança": "ELE-PES-BALANCA",
    "Barra De Pinos Fêmea 2x20": "ELE-BPN-F2X20",
    "Barra Pino Fêmea 1x5 90 graus 2.54mm": "ELE-BPN-F1X5",
    "Barra de Pinos 16 Vias (1x16) Simples MCI Fêmea": "ELE-BPN-F1X16",
    "Barra pino macho 1x40 90 graus": "ELE-BPN-M1X40",
    "Botão Pulsador 22mm Retentivo Verde 1na Xa2eh031 Schneider": "ELE-CAB-BOTAO",
    "Buzzer 3.3v com Oscilador interno": "ELE-CMP-BUZZER",
    "CI 74HC595": "ELE-CTR-74HC595",
    "Cabo De Energia Para Fonte, 1.50M, 3x0,75, 10Ah - Nbr14136": "ELE-CAB-FORCA15M",
    "Cabo De Rede Cat.5e": "ELE-CAB-CAT5E",
    "Cabo Extensor Usb 2.0 Cristal 3m Fce3 Feasso (resistente)": "ELE-CAB-USB3M",
    "Cabo Hdmi Curvo 1 metro": "ELE-CAB-HDMI",
    "Cabo Manga 4X26 Awg Blindado Malha de Cobre Verde/Palha": "ELE-CAB-MANGA4X26",
    "Caixa Patola CPU": "EST-CAS-PBT114",
    "Caixa Patola Sensor": "EST-CAS-PBT075",
    "Caixa de papelão V2": "EMB-EMB-CAIXA",
    "Capa para Conector DB9": "ELE-CAB-CAPADB9",
    "Capacitor Eletrolítico 16v 470uF": "ELE-CAP-470UF16V",
    "Cartão de memória 32GB /16GB (070341075586) (070341075593)": "ELE-CTR-SD32GB",
    "Chapa Fixação LCD / Shield Raspberry": "EST-MET-CHAPAFIX",
    "Conector DB9 Serial RS232 Macho 180º": "ELE-CAB-DB9M",
    "Conector Modu Femea 5 Vias": "ELE-CAB-ALOJMODU",
    "DB9 fêmea 90 graus para placa": "ELE-CMP-DB9F",
    "Diodo 1N4007 Retificador 1A 1000V (DO-41)": "ELE-DIO-1N4007",
    "Diodo 1N4148 Comutação Rápida 0,15A 100V (DO-35)": "ELE-DIO-1N4148",
    "Display LCD 20×4 Backlight Azul": "ELE-CMP-LCD20X4",
    "Fita Cristal (Durex)": "EMB-EMB-FITACRISTAL",
    "Fita dupla face 3M (1/4 ROLO)": "EST-PAR-DUPLAFACE",
    "Fitilho (1/8)": "EMB-EMB-FITILHO",
    "Fonte MW (5v 3A - 0 - 12v 2A) [V1 - GND - V2]": "ELE-CAB-FONTEMW",
    "Fusível de Vidro 5x20mm 3A 250V": "ELE-CMP-FUS3A",
    "Led Difuso 3mm - Verde": "ELE-LED-VERDE",
    "Led Difuso 3mm - Vermelho": "ELE-LED-VERMELHO",
    "Led RGB Tricolor 5mm com 4 Terminais - 3.000 MCD, catodo comum": "ELE-LED-RGB",
    "Microcontrolador ATTINY85": "ELE-CTR-ATTINY85",
    "Monitor 15\" LED HD 60Hz": "ELE-PER-MONITOR15",
    "Módulo Conversor RS232 para TTL MAX3232": "ELE-CTR-MAX3232",
    "Organizador De Fios E Cabos Espiral 1/2": "EST-ACA-ORGFIO",
    "Parafuso AA PN PH ZB 2,9x6,5 (Chapa metal Shield Rapsberry)": "EST-PAR-AA29X65",
    "Parafuso AA PN PH ZB 4,2x9,5 (Fixar patola sensor)": "EST-PAR-AA42X95",
    "Parafuso Fixes CH 3,5x16 - (Fechar CPU)": "EST-PAR-CH35X16",
    "Parafuso Frances ZB 3/8x21/2 (Estrutura Tubo)": "EST-PAR-FR38X212",
    "Parafuso Frances ZB 3/8x4\" (Estrutura Tubo)": "EST-PAR-FR38X4",
    "Parafuso Frances ZB 3/8x5\" (Estrutura Tubo)": "EST-PAR-FR38X5",
    "Parafuso PN PH 3x6 M3 (Parafuso Fonte)": "EST-PAR-M3X6",
    "Parafuso PN PH 4x10": "EST-PAR-M4X10",
    "Parafuso PN PH 4x20": "EST-PAR-M4X20",
    "Parafuso PN PH 6x12 (M6x12) (Display Balança)": "EST-PAR-M6X12",
    "Parafuso Sextavado ZB RI 10x50 (Fixa Balança)": "EST-PAR-SEX10X50",
    "Plug RJ45 Macho Furukawa SohoPlus CAT5e": "ELE-CAB-RJ45M",
    "Plástico Bolha": "EMB-EMB-BOLHA",
    "Plástico Strach (1/3)": "EMB-EMB-STRETCH",
    "Porca rebite M4": "EST-PAR-REBROSCM4",
    "Porta Fusível 5x20 AS-06": "ELE-CMP-PORTAFUS",
    "RJ45 fêmea 90 graus para placa": "ELE-CMP-RJ45F",
    "Raspberry pi3 B+": "ELE-CTR-RPI3B",
    "Rebite Pop 4,0x10": "EST-PAR-REB4X10",
    "Rebite Pop 4,0x12": "EST-PAR-REB4X12",
    "Rebite Pop 4,0x8": "EST-PAR-REB4X8",
    "Relé Songle 5V": "ELE-CMP-RELE5V",
    "Resistor 10 Ω": "ELE-RES-10",
    "Resistor 100 Ω": "ELE-RES-100",
    "Resistor 100k Ω": "ELE-RES-100K",
    "Resistor 10k Ω": "ELE-RES-10K",
    "Resistor 150 Ω": "ELE-RES-150",
    "Resistor 1k Ω": "ELE-RES-1K",
    "Resistor 1k2 Ω": "ELE-RES-1K2",
    "Resistor 220 Ω": "ELE-RES-220",
    "Resistor 270Ω": "ELE-RES-270",
    "Resistor 2k2 Ω": "ELE-RES-2K2",
    "Resistor 390 Ω": "ELE-RES-390",
    "Resistor 3k9 Ω": "ELE-RES-3K9",
    "Rodízio 2'' rosca 3/8": "EST-PAR-RODIZIO2",
    "Sensor Temperatura LM35DZ": "ELE-TRS-LM35",
    "Sensores ultrassônicos HYSR05": "ELE-CMP-HYSR05",
    "Shield Raspberry": "ELE-PCB-SHIELDRPI",
    "Shield Sensor": "ELE-PCB-SHIELDSENS",
    "Tapa furo 40x40": "EST-ACA-TAPA40X40",
    "Tapa furo 60x40": "EST-ACA-TAPA60X40",
    "Tblock 3 Vias 90 graus femea para fio": "ELE-CMP-TBLOCK3F",
    "Tblock 3 Vias 90 graus macho para placa": "ELE-CMP-TBLOCK3M",
    "Terminal Forquilha Azul Para Fios De 1,5 A 2,5Mm": "ELE-CAB-FORQUILHA",
    "Terminal Modu 1mm": "ELE-CAB-TERMMODU",
    "Tomada AS-03 Tripolar com Chave Gangorra": "ELE-CAB-TOMADAAS03",
    "Transistor NPN BC337": "ELE-TRS-BC337",
    "Transistor PNP BC327": "ELE-TRS-BC327",
    "Trimpot 3386H - 5K": "ELE-CMP-TRIMPOT5K",
}

# material do Controle -> SKU NOVO (grupo, classe)
GRUPOS = {"ELE": "ELÉTRICA / ELETRÔNICA", "EST": "ESTRUTURAL", "EMB": "EMBALAGENS / OUTROS"}
CLASSES = {
    "SUB": "SUBCONJUNTOS", "MET": "METALÚRGICO", "ADE": "ADESIVOS", "PAR": "PARAFUSOS",
    "CAB": "CABEAMENTO", "CTR": "CONTROLADORES", "CMP": "OUTROS COMPONENTES", "PER": "ADICIONAL CORREIOS",
    "CAS": "CASES", "ACA": "ACABAMENTO", "BPN": "BARRA DE PINO", "PCB": "PCB",
}
NOVOS = {
    "CPU": "ELE-SUB-CPUV2",
    "PCB BoxCubo v2.5": "ELE-PCB-PCBV25",
    "Adapdator eixo giratorio EYE": "EST-MET-EYEADAPTEIXO",
    "Base Eye": "EST-MET-EYEBASE",
    "Tampa da Base Eye": "EST-MET-EYETAMPA",
    "Braço 1 Eye": "EST-MET-EYEBRACO1",
    "Braço 2 EYE": "EST-MET-EYEBRACO2",
    "Braço 3 EYE": "EST-MET-EYEBRACO3",
    "Braço 4 EYE": "EST-MET-EYEBRACO4",
    "Junta U EYE": "EST-MET-EYEJUNTAU",
    "Suporte monitor EYE": "EST-MET-EYESUPMON",
    "Case Raspberry Pi 3 B+": "EST-CAS-CASERPI",
    "Adesivo CPU V2": "EST-ADE-CPUV2",
    "Adesivo Logo Box Cubo EYE": "EST-ADE-LOGOEYE",
    "Adesivo Logo Box Cubo V2": "EST-ADE-LOGOV2",
    "Adesivo Logo Hexágono": "EST-ADE-HEXAGONO",
    "Adesivo Logo Vision": "EST-ADE-LOGOVISION",
    "Arruela Plastica 1/4": "EST-PAR-ARRPL14",
    "Arruela M5": "EST-PAR-ARRM5",
    "Arruela de borracha 1/8": "EST-PAR-ARRBOR18",
    "Manipulo M6": "EST-PAR-MANIPM6",
    "Parafuso Auto Brocante Sextavado 5,5x19mm (12x3/4)": "EST-PAR-AB55X19",
    "Parafuso Fixer CH 3,5x16 (Fechar CPU)": "EST-PAR-CH35X16FX",
    "Parafuso PN PH 4x25 (M4x25)": "EST-PAR-M4X25",
    "Parafuso PN PH 6x10 (M6x10)": "EST-PAR-M6X10",
    "Parafuso PN PH 6x40 (M6x40)": "EST-PAR-M6X40",
    "Parafuso PN PH 6x8 (M6x8)": "EST-PAR-M6X8",
    "Porca rebite M6": "EST-PAR-REBROSCM6",
    "Perfil Alpo2 Sobrepor Preto": "EST-ACA-PERFILALPO2",
    "Silicone Neutro Incolor": "EST-ACA-SILICONE",
    "Barra pino macho 1x40 180 graus": "ELE-BPN-M1X40R",
    "Botão 19mm Inox Lig/desl. Start Led C/ Trava + Chicote": "ELE-CAB-BOTAO19",
    "Cabo Carregamento V8 Micro Usb 6920231021077": "ELE-CAB-V8MICROUSB",
    "Cabo de Alimentação da Balança": "ELE-CAB-ALIMBAL",
    "Cabo de Força Tripolar 3 Metros": "ELE-CAB-FORCA3M",
    "Extensao Eletrica Tripolar 3Mts 3 Entradas": "ELE-CAB-EXTENSAO3M",
    "Fonte Chaveada 25W 5A RS-25-5 88-264V Mean Well AC/DC Entrada Automática": "ELE-CAB-FONTE5V",
    "Fonte de Alimentação RS-25-12 12V 2,1A 25W": "ELE-CAB-FONTE12V",
    "Termo Retrátil 4,8mm": "ELE-CAB-TERMO48",
    "Conversor de USB para 1 saída serial RS232": "ELE-CTR-USBRS232",
    "Mini Pc Nuc Intel": "ELE-CTR-NUC",
    "Fita led 12v": "ELE-CMP-FITALED12V",
    "Webcam 4k ??? kkk": "ELE-PER-WEBCAM4K",
}
# nomes mais claros para os novos (o original fica guardado em "nome_controle")
NOMES = {
    "CPU": "CPU V2 (Raspberry + Shield + display)",
    "Webcam 4k ??? kkk": "Webcam 4K",
    "Arruela Plastica 1/4": "Arruela plástica 1/4",
}
UNIDADES = {"un": "und", "Metro": "m", "Metros": "m", "m": "m", "2 un": "pct", "10 un": "pct"}
# Peças que trocam de unidade (o sistema SKU usava rolo; o Controle usa metro)
TROCA_UNIDADE = {"ELE-CAB-MANGA4X26": "m"}


def main():
    mats = json.loads((AQUI / "materiais.json").read_text(encoding="utf-8"))
    mapa = {norm(k): v for k, v in MAPA.items()}
    novos = {norm(k): v for k, v in NOVOS.items()}
    materiais = []
    por_nome = {}
    for nome, und, est, cheio, preco, _link, fab in mats:
        n = norm(nome)
        if n in mapa:
            sku, novo = mapa[n], False
        elif n in novos:
            sku, novo = novos[n], True
        else:
            raise SystemExit(f"material sem SKU: {nome!r}")
        g, c, _ = sku.split("-", 2)
        unidade = UNIDADES[und]
        item = {
            "nome_controle": n, "sku": sku, "novo": novo,
            "nome": NOMES.get(n, n), "grupo": GRUPOS[g], "classe": CLASSES.get(c, c),
            "unidade": TROCA_UNIDADE.get(sku, unidade), "troca_unidade": sku in TROCA_UNIDADE,
            "tipo": "consumivel" if unidade == "m" else "peca",
            "estoque": est, "preco": preco, "fabricado": bool(fab),
        }
        materiais.append(item)
        por_nome[n] = sku
    skus = [m["sku"] for m in materiais]
    assert len(set(skus)) == len(skus), "SKU repetido"

    receitas, atual = {}, None
    ids = {"V2": "v2", "EYE": "eye", "VISION": "vision",
           "SUB:CPU": "ELE-SUB-CPUV2", "SUB:Shield Raspberry": "ELE-PCB-SHIELDRPI"}
    for linha in (AQUI / "receitas.txt").read_text(encoding="utf-8").splitlines():
        if linha.startswith("##"):
            atual = ids[linha[2:]]
            receitas[atual] = []
            continue
        nome, qtd = linha.rsplit("|", 1)
        sku = por_nome[norm(nome)]
        receitas[atual].append([sku, float(qtd)])
    # a V2 do Controle usa o CPU como peça pronta (fabricada em separado)
    receitas["v2"].append(["ELE-SUB-CPUV2", 1.0])

    out = {"origem": "Controle BoxCubo (192.168.15.10/materiais)", "data": "09/11/2026",
           "materiais": materiais, "receitas": receitas}
    dst = ROOT / "backend" / "internal" / "store" / "receitas_controle.json"
    dst.write_text(json.dumps(out, ensure_ascii=False, indent=1), encoding="utf-8")
    print(f"{len(materiais)} materiais ({sum(m['novo'] for m in materiais)} novos), "
          + ", ".join(f"{k}: {len(v)}" for k, v in receitas.items()))


if __name__ == "__main__":
    main()
