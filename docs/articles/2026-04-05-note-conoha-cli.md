# CLIひとつでVPSデプロイ完了 — conoha-cliとClaude Code Skillで変わるインフラ構築

## はじめに

VPSでアプリケーションを動かそうとすると、意外とやることが多い。まず管理画面にログインしてサーバーを作り、SSHキーを設定し、ターミナルからサーバーに入って、OSのアップデートをかけ、Dockerをインストールし、Composeも入れて、ソースコードを転送して、ようやくビルドとデプロイ。ここまでたどり着くまでに何度もブラウザとターミナルを行き来することになる。

もちろん、TerraformやAnsibleのようなIaCツールを使えばこの手順を自動化できる。しかし、個人開発や小規模なプロジェクトでは、これらのツールを導入するコスト自体がオーバーヘッドになりがちだ。HCLやYAMLのPlaybookを書き、ステートファイルを管理し、ディレクトリ構成を整え…。やりたいことは「自分のアプリをサーバーで動かす」だけなのに、そこに至るまでの道のりが長い。

この記事では、そんな「ちょうどいいインフラ管理」を実現するCLIツール **conoha-cli** を紹介する。ConoHa VPS3のAPIに対応したGo製のコマンドラインツールで、サーバーの作成からアプリのデプロイ、運用管理までをターミナルから一貫して操作できる。さらに、Claude Codeの **skill** を導入すれば、自然言語でインフラ構築を指示することも可能だ。

前半ではconoha-cliの概要とインストール方法を紹介し、後半では実際に筆者がサーバーを立ててアプリをデプロイした体験と、同じことをClaude Code Skillで自然言語から実行した体験を書いていく。CLIに慣れている人も、これからVPSを触ってみたい人も、AIツールに興味がある人も、それぞれの視点で楽しめる内容になっていると思う。

## conoha-cliとは

conoha-cliは、ConoHa VPS3のAPIをターミナルから操作するためのCLIツールだ。Goで書かれたシングルバイナリとして配布されているため、インストールはファイルをひとつ置くだけで完了する。macOS、Linux、Windowsに対応しており、環境を選ばない。

認証まわりは、GitHub CLIの`gh auth`に似た設計になっている。`conoha auth login`を実行するとテナントID・ユーザー名・パスワードを聞かれるので、ConoHaの管理画面で発行した認証情報を入力すればよい。認証トークンはローカルに保存され、有効期限が切れる5分前に自動的に再認証が走るため、使っている途中でトークン切れに悩まされることがない。複数のConoHaアカウントを使っている場合は、`conoha auth switch`でプロファイルを切り替えられるので、検証環境と本番環境を分けて管理するのも簡単だ。

出力フォーマットはデフォルトのテーブル表示のほか、JSON、YAML、CSVを選べる。たとえば`conoha server list --format json`とすればJSON形式で結果が返ってくるので、`jq`と組み合わせてスクリプトに組み込むことも容易だ。`--no-input`フラグをつければ対話プロンプトをすべて抑制でき、CIパイプラインやAIエージェントからの自動実行にも対応する。

インストール方法はいくつか用意されている。macOSやLinuxならHomebrewが最も手軽だ。

```bash
brew install crowdy/tap/conoha
```

WindowsならScoopに対応している。

```powershell
scoop bucket add crowdy https://github.com/crowdy/crowdy-bucket
scoop install conoha
```

Goの開発環境がある場合は、`go install`でもインストールできる。

```bash
go install github.com/crowdy/conoha-cli@latest
```

GitHub Releasesページからバイナリを直接ダウンロードする方法もある。いずれの方法でも、インストール後に`conoha auth login`で認証を通せば準備完了だ。

```bash
conoha auth login
# テナントID、ユーザー名、パスワードを入力
```

ここまででセットアップは終わり。あとはターミナルからConoHaのリソースを自由に操作できる。

## 体験記: CLIでサーバーを立ててアプリをデプロイしてみた

ここからは、実際に筆者がconoha-cliを使ってゼロからアプリをデプロイした体験を書いていく。

### サーバーを作る

まず、どのスペックのサーバーを作るか決めるために、フレーバー（プラン）の一覧を確認してみた。

```bash
conoha flavor list
```

テーブル形式で利用可能なプランがずらっと表示される。vCPU数やメモリ、月額料金が一覧できるので、管理画面を開かなくてもプラン選びができる。今回は検証用なので、最小構成の`g2l-t-c2m1`（2vCPU / 1GB RAM）を選ぶことにした。

次にOSイメージを確認する。

```bash
conoha image list
```

Ubuntu、AlmaLinux、その他さまざまなOSイメージが並んでいる。ここではUbuntu 24.04を使うことにした。

SSHキーペアも必要だ。まだ登録していなかったので、ここで作成しておく。

```bash
conoha keypair create --name my-key
```

秘密鍵が標準出力に表示されるので、ファイルに保存しておく。これでサーバー作成の材料が揃った。

```bash
conoha server create --name my-app \
  --flavor g2l-t-c2m1 \
  --image <ubuntu-image-id> \
  --key-name my-key \
  --wait
```

`--wait`フラグをつけると、サーバーがACTIVE状態になるまでコマンドが待機してくれる。ターミナルでプログレスが表示され、1〜2分ほどでサーバーが立ち上がった。これは地味にありがたい。`--wait`なしだと即座にコマンドが返ってきて、`conoha server list`で自分でステータスを確認しに行く必要があるが、`--wait`をつければ「コマンドが返ってきた＝サーバーが使える」という状態になる。

サーバーが立ち上がったら、SSHで接続してみる。

```bash
conoha server ssh my-app
```

ここで便利だと感じたのが、サーバーIDではなく名前で指定できること、そしてSSHキーの自動解決だ。キーペアの名前からローカルの秘密鍵パスを推測してくれるので、`-i`オプションでいちいちパスを指定する必要がない。普通にサーバーに入れた。

### アプリをデプロイする

SSHで入れることは確認できたが、ここからDockerを入れてアプリを動かすまでが本来は面倒なところだ。ところが、conoha-cliにはそれを一発で片付ける`app`コマンドが用意されている。

まず、サーバーをアプリデプロイ用に初期化する。

```bash
conoha app init my-app --app-name myapp
```

このコマンドひとつで、サーバー上にDockerとDocker Composeがインストールされ、`/opt/conoha/myapp/`にワーキングディレクトリと、`/opt/conoha/myapp.git/`にbareリポジトリが作られる。gitのpost-receiveフックも自動設定されるので、gitプッシュによるデプロイ環境まで整う。手動でやったら15分はかかりそうな作業がワンコマンドだ。

次に、デプロイするアプリを用意する。今回はシンプルなExpress.jsアプリを使った。まず`Dockerfile`を書く。

```dockerfile
FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --production
COPY . .
EXPOSE 3000
CMD ["node", "index.js"]
```

そして`docker-compose.yml`はこうなる。

```yaml
services:
  web:
    build: .
    ports:
      - "3000:3000"
    restart: unless-stopped
```

ローカルにこの2つのファイルとアプリのソースコードがある状態で、デプロイコマンドを実行する。

```bash
conoha app deploy my-app --app-name myapp
```

実行すると、ローカルのカレントディレクトリがtarアーカイブとしてパッケージングされ、SSH経由でサーバーに転送される。`.dockerignore`に書いたパターンと`.git/`ディレクトリは自動的に除外してくれる。サーバー側では`docker compose up -d --build`が走り、コンテナのビルドと起動が行われる。

デプロイが完了したら、状態を確認してみる。

```bash
conoha app status my-app --app-name myapp
```

コンテナがRunning状態になっているのが見える。ログも確認してみよう。

```bash
conoha app logs my-app --app-name myapp --follow
```

`--follow`をつけるとリアルタイムにログがストリーミングされてくる。Expressのアクセスログが流れてきて、ちゃんと動いているのがわかる。ブラウザからサーバーのIPアドレスの3000番ポートにアクセスすると、アプリが表示された。

ここまでの手順を振り返ると、`server create` → `app init` → `app deploy`の3コマンドで、サーバーの作成からアプリの稼働まで到達している。管理画面とターミナルを行き来する必要もなく、Dockerのインストール手順を調べる必要もなかった。

### 運用もCLIで

デプロイした後の運用もコマンドで完結する。たとえばデータベースの接続先を環境変数として設定したくなったら、こうする。

```bash
conoha app env set my-app --app-name myapp DATABASE_URL=postgres://user:pass@db:5432/mydb
```

環境変数はサーバー上の`.env.server`ファイルに永続化される。次に`app deploy`を実行した際に自動的に`.env`としてコピーされるので、再デプロイ時にも設定が引き継がれる仕組みだ。設定を変更したら再起動する。

```bash
conoha app restart my-app --app-name myapp
```

サーバー上に複数のアプリをデプロイしている場合は、`app list`で一覧を確認できる。

```bash
conoha app list my-app
```

アプリ名と状態（running / stopped）が表示される。不要になったアプリは`app destroy`で後片付けまでやってくれる。

```bash
conoha app destroy my-app --app-name myapp
```

コンテナの停止、ワーキングディレクトリの削除、gitリポジトリの削除、環境変数ファイルの削除まで一括で行われる。確認プロンプトが出るので、うっかり消してしまう心配もない。

## 体験記: Skillを入れて自然言語でやり直してみた

前のセクションで手動で行ったサーバー作成からアプリデプロイまでの流れを、今度はClaude Codeに頼んでやってみることにした。

### Skillをインストールする

conoha-cliにはClaude Code用の **skill** が用意されている。skillとは、Claude Codeに特定のドメイン知識を教えるためのプラグインのようなものだ。通常のClaude Codeはプログラミング全般の知識は持っているが、「ConoHa VPSでサーバーを作るにはどのAPIを叩けばよいか」「Dockerをインストールするためにどのコマンドを実行するか」といったインフラ構築の具体的な手順までは知らない。skillをインストールすることで、Claude Codeがこうしたドメイン固有の知識を持てるようになる。

インストールは一行で終わる。

```bash
conoha skill install
```

これだけで、`~/.claude/skills/conoha-cli-skill/`にskillファイルがダウンロードされる。skillの中身はMarkdownで書かれたレシピ集で、Docker Composeアプリのデプロイ手順、カスタムスクリプトでのサーバー構築、Kubernetesクラスターのセットアップ、OpenStack環境の構築、Slurm HPCクラスターの構築など、さまざまなインフラ構築パターンが収録されている。

更新も簡単で、`conoha skill update`で最新版に同期される。不要になったら`conoha skill remove`で削除できる。

### 自然言語でデプロイしてみる

skillをインストールした状態でClaude Codeを起動し、こう指示してみた。

```
> ConoHaにUbuntuサーバーを作って、Expressアプリをデプロイして
```

するとClaude Codeが応答し、まずフレーバーの確認から始まった。

```
Claude: ConoHaにサーバーを作成してアプリをデプロイしますね。
        まず利用可能なフレーバーを確認します。
        （conoha flavor list を実行）
        いくつかのプランがあります。g2l-t-c2m1（2vCPU / 1GB）で
        よろしいですか？
```

「はい」と答えると、次にイメージの選択に進んだ。Ubuntu 24.04を選び、キーペアも指定すると、Claude Codeがサーバー作成コマンドを組み立てて実行した。

```
Claude: サーバーを作成しています...
        （conoha server create --name express-app --flavor g2l-t-c2m1
         --image <id> --key-name my-key --wait を実行）
        サーバーが起動しました。Docker環境を初期化します。
        （conoha app init express-app --app-name myapp を実行）
        初期化が完了しました。コードをデプロイします。
        （conoha app deploy express-app --app-name myapp を実行）
        デプロイが完了しました！
        http://xxx.xxx.xxx.xxx:3000 でアクセスできます。
```

前のセクションで筆者が手動で行った手順——フレーバーの確認、サーバー作成、app init、app deploy——が、数回の対話のやりとりで完了してしまった。コマンドのフラグや引数を覚える必要もなく、「やりたいこと」を伝えるだけでClaude Codeがskillのレシピに沿って適切なコマンドを組み立ててくれる。

重要なのは、skillの裏側で実行されているのはconoha-cliのコマンドそのものだということだ。ブラックボックスではない。Claude Codeが何を実行しているかはターミナルにすべて表示されるし、途中で「ちょっと待って、フレーバーはもう少し大きいのにして」と言えば柔軟に対応してくれる。CLIの操作を理解している人にとっては、自分の代わりにコマンドを叩いてくれるアシスタントのようなものだ。

### 別のレシピも試してみる

skillにはDocker Composeアプリ以外のレシピも含まれている。試しにKubernetesクラスターの構築を頼んでみた。

```
> ConoHaでk3sのKubernetesクラスターを作って
```

すると今度は別のレシピが読み込まれ、マスターノードとワーカーノードの台数を聞かれた。レシピが変わるとアプローチ自体も変わり、サーバーを複数台作成して、k3sのインストールスクリプトを順番に実行していく流れになった。先ほどのDocker Composeデプロイとはまったく異なる手順だが、同じように対話形式で進んでいく。

用意されているレシピは現在5種類ある。

- **Docker Composeアプリ** — 先ほどのデモで使ったパターン。docker-compose.ymlがあるプロジェクトを手軽にデプロイする
- **カスタムスクリプト** — 任意のシェルスクリプトでサーバーをプロビジョニングする
- **Kubernetes (k3s)** — 軽量Kubernetesクラスターを構築する
- **OpenStack (DevStack)** — 開発用のOpenStack環境を立ち上げる
- **Slurm HPC** — 高性能計算用のジョブスケジューラ環境を構築する

skillはGitHubリポジトリとして公開されているので、既存のレシピを参考にして自分のユースケースに合わせたレシピを追加することもできる。Markdownで手順を書くだけなので、プログラミングの知識は必要ない。

### 手動でやる意味、skillでやる意味

ここまで体験してみて感じたのは、手動操作とskillはどちらかが上位互換というわけではなく、使い分けるものだということだ。

CLIで手動操作するのは、細かい制御が必要なときや、コマンドの動作を理解したいときに適している。セキュリティグループの設定をひとつずつ確認しながら追加したいとか、デプロイのプロセスを理解しておきたいとか、そういう場面では手動のほうがいい。

skillを使うのは、定型的な構築作業を素早く済ませたいときだ。「検証用にさっとサーバーを立ててアプリを動かしたい」という場面では、フレーバーのIDを調べたりフラグの名前を思い出したりする手間を省いて、やりたいことだけ伝えればよい。

そして、この2つは矛盾しない。手動でCLIを使い込んで操作を理解している人ほど、skillが裏で何をしているかがわかるので安心して使える。逆にskillから入った人も、Claude Codeが実行するコマンドを見ることで、CLIの使い方を自然に学んでいける。

## まとめ

この記事では、conoha-cliを使ったVPSの管理とアプリのデプロイを紹介した。

ターミナルからサーバーの作成、Docker環境の構築、アプリのデプロイ、ログの確認、環境変数の管理まで一貫して操作でき、管理画面とターミナルを行き来する必要がなくなる。`docker-compose.yml`があるプロジェクトなら、`app init` → `app deploy`の2コマンドでサーバー上にアプリが立ち上がる。

さらにClaude Code Skillを導入すれば、同じ操作を自然言語で指示できるようになる。skillはconoha-cliのコマンドを内部で使っているため、何が実行されているかは常に透明だ。ブラックボックスにインフラを任せる不安感はなく、CLIの知識がそのまま活きる。

conoha-cliの設計には、AIエージェントとの親和性が随所に組み込まれている。`--no-input`フラグによる非対話実行、`--format json`による構造化出力、明確に定義された終了コード。これらはCIパイプラインやスクリプトからの利用だけでなく、Claude CodeのようなAIエージェントが確実にコマンドの結果を解釈するための基盤でもある。

conoha-cliはオープンソースとして開発されており、コントリビューションを歓迎している。興味があればぜひ試してみてほしい。

**インストール:**

```bash
brew install crowdy/tap/conoha
```

**リポジトリ:**

- conoha-cli: https://github.com/crowdy/conoha-cli
- conoha-cli-skill: https://github.com/crowdy/conoha-cli-skill
- ドキュメントサイト: https://crowdy.github.io/conoha-cli-pages/
