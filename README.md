Cyber Arena (Cheesy Arena Lite fork for 4 teams per match tournament)
============
A game-agnostic field management system that just works.

For the game-specific version, see [Cheesy Arena](https://github.com/Team254/cheesy-arena).

## License
Teams may use Cheesy Arena Lite freely for practice, scrimmages, and off-season events. See [LICENSE](LICENSE) for more details.

## Installing
**From a pre-built release**

Download the [latest release](https://github.com/Team254/cheesy-arena-lite/releases). Pre-built packages are available for Linux, macOS (x64 and M1), and Windows.

On recent versions of macOS, you may be prevented from running an app from an unidentified developer; see [these instructions](https://support.apple.com/guide/mac-help/open-a-mac-app-from-an-unidentified-developer-mh40616/mac) on how to bypass the warning.

**From source**

1. Download [Go](https://golang.org/dl/) (version 1.16 or later recommended)
1. Clone this GitHub repository to a location of your choice
1. Navigate to the repository's directory in the terminal
1. Compile the code with `go build`
1. Run the `cheesy-arena-lite` or `cheesy-arena-lite.exe` binary
1. Navigate to http://localhost:8080 in your browser (Google Chrome recommended)

**IP address configuration**

When running Cheesy Arena Lite on a playing field with robots, set the IP address of the computer running Cheesy Arena Lite to 10.0.100.5. By a convention baked into the FRC Driver Station software, driver stations will broadcast their presence on the network to this hardcoded address so that the FMS does not need to discover them by some other method.

When running Cheesy Arena Lite without robots for testing or development, any IP address can be used.

## Os três modos (branch `vernum`)

O binário lê o modo do ambiente. **Sem variável nenhuma ele é o de sempre** — e é assim que se roda um
evento que não tem nada a ver com o Vernum.

| `ARENA_MODE` | Cabeçalho | Login | Campo, PLC, driver stations |
|---|---|---|---|
| *(vazio)* / `standalone` | roxo | `admin` + a senha das configurações | ligados |
| `local` | âmbar, selo **Local** | `local`/`local` **só do próprio computador**; de fora, a senha das configurações | ligados |
| `cloud` | verde-azulado, selo **Nuvem** | só o Vernum, e o papel vem do evento | desligados |

### Variáveis

| Variável | Padrão | Para quê |
|---|---|---|
| `ARENA_MODE` | `standalone` | o modo acima |
| `ARENA_PORT` | `9080` | porta HTTP |
| `ARENA_DB_PATH` | `./event.db` | onde fica o banco |
| `ARENA_BASE_PATH` | *(vazio)* | prefixo HTTP, quando duas arenas dividem um host |
| `ARENA_MASTER_URL` | *(vazio)* | endereço do Vernum Arena Master |
| `ARENA_TOKEN` | *(vazio)* | token desta instância; vazio, nenhuma chamada de rede sai |
| `ARENA_SYNC_SECONDS` | `30` | de quanto em quanto tempo mandar |
| `ARENA_VENUE_SLOT`, `ARENA_VENUE_LABEL` | *(vazio)* | qual mesa ou arena este processo é; em nuvem quem responde é o provisionador |
| `ARENA_FLL_ROLE`, `ARENA_FLL_KEY`, `ARENA_FLL_MASTER_URL`, `ARENA_FLL_CLIENTS` | *(vazio)* | a malha da FLL, fiada pelo Arena Master no cluster |
| `VERNUM_SSO_URL`, `VERNUM_API_URL`, `VERNUM_CLIENT_ID`, `VERNUM_CLIENT_SECRET` | *(vazio)* | o login do Vernum, em modo nuvem |

`ARENA_FLL_MASTER_URL` é a **base** da API do mestre (`http://mestre/api/fll`), não a URL das
pontuações: quem chama acrescenta `/scores`, `/review` ou `/start-match`.

### Temporadas

Uma temporada é um **documento JSON** em `game/seasons/`, compilado no binário por `go:embed`. O que
fica gravado numa partida é o **tally** — contagens por ação e ocupação de slot, com o período que
ocupou cada um —, nunca os pontos: pontos, categorias, ranking points, elegibilidade e vetor de
desempate são sempre derivados do tally mais o documento mais o nível do evento. É isso que faz um
Team Update publicado em abril poder ser aplicado com a temporada quase toda jogada.

Duas consequências que valem estar escritas:

- **O par `occupiedIn`/`everIn`.** Os pontos leem quem ocupa o slot agora; o AUTO RP lê quem *já*
  ocupou no autônomo. O coral retirado no teleop perde os pontos e mantém o RP, que é exatamente a
  regra 6.5.1 e exatamente o que o sistema oficial da FIRST errou em 2025 (Team Update 20).
- **Um slot por robô** (`"slots": "robots"`) faz "cada robô recebe um único crédito" ser impossível
  de violar, em vez de um teto numérico que um clique a mais fura.

Cada documento carrega os próprios **casos de teste**, e eles são portão: `go test ./game/...` roda
todos os casos de todas as temporadas embutidas, e o validador exige que cada ranking point apareça
verdadeiro em um caso e falso em outro. Uma temporada cujo RP nunca dispara é indistinguível de uma
que funciona — até um domingo à tarde.

### A arbitragem sai do documento

A tela de arbitragem não tem mais bloco de HTML por ação: ela é desenhada de `game/seasons/*.json`
pelo `static/js/season_panel.js`. Um clique vira **uma** mensagem — `scoreTally` — e a guarda de
estado da partida mora nela, num lugar só; antes havia quatro caminhos que escreviam pontos, em duas
cópias, e só um deles recusava clique antes da partida começar. O painel não sabe quanto vale nada:
manda o clique e o servidor responde com o placar que calculou, as categorias vivas e quais ranking
points já estão ganhos.

O documento chega pela mensagem `matchLoad` do websocket, nunca pelo template: o `html/template`
escaparia o JSON como literal de string, e por chegar assim uma troca de temporada em Configurações
recarrega todos os painéis sozinha. Recarregar a página no meio da partida devolve o estado inteiro,
porque ele é do servidor e não do navegador.

Um resultado gravado **antes** das temporadas não tem tally, e continua sendo lido como sempre: a
soma dos quatro números digitados. As quatro colunas mantiveram o nome no banco e na API; só os
campos Go viraram `Legacy*`. Nada foi reescrito no banco de um evento que já aconteceu.

### Conectar a um evento

**Configurações → Arena Master** conduz o processo: define a senha desta arena, cola o token que o
administrador do evento mandou, confere os dados com o servidor, escolhe qual mesa ou arena é esta
máquina, registra e importa a lista de equipes. Nada é enviado antes de alguém conferir na tela.

Depois disso a mesma página vira o painel: o que já foi enviado, o último erro em português e um
botão de sincronizar agora, para o evento que roda sem rede.

### Imagem

Só para o provisionamento em nuvem, feito pelo Arena Master:

```bash
docker build --build-arg VERSION=$(git describe --tags --always) -t cyber-arena .
docker run -p 9080:9080 -v arena-data:/data -e ARENA_MODE=cloud cyber-arena
```

## Further reading
Please see the game-specific [Cheesy Arena](https://github.com/Team254/cheesy-arena) README for technical details and acknowledgements.
