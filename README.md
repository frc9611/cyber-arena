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
| `VERNUM_SSO_URL`, `VERNUM_API_URL`, `VERNUM_CLIENT_ID`, `VERNUM_CLIENT_SECRET` | *(vazio)* | o login do Vernum, em modo nuvem |

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
