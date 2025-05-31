# Create Index Tool

このツールは Warren アプリケーション用の Firestore インデックスを作成するためのものです。

## 必要な環境

- Go 1.22+
- gcloud CLI がインストールされ、適切に認証されていること

## 使用方法

### インストール

```bash
cd resources/create_index
go mod tidy
go build -o create_index
```

### 実行

#### Dry Run（実際にはインデックスを作成せず、チェックのみ）

```bash
./create_index create --dry-run \
  --firestore-project-id=your-project-id \
  --firestore-database-id=your-database-id
```

#### インデックスの作成

```bash
./create_index create \
  --firestore-project-id=your-project-id \
  --firestore-database-id=your-database-id
```

### 環境変数

Warren の設定に合わせて、以下の環境変数を使用できます：

- `WARREN_FIRESTORE_PROJECT_ID`: Firestore プロジェクト ID
- `WARREN_FIRESTORE_DATABASE_ID`: Firestore データベース ID（デフォルト: "(default)"）

環境変数を設定した場合：

```bash
export WARREN_FIRESTORE_PROJECT_ID=your-project-id
export WARREN_FIRESTORE_DATABASE_ID=your-database-id
./create_index create --dry-run
```

## 作成されるインデックス

このツールは以下のコレクション用のインデックスを作成します：

- `alerts`
- `tickets`
- `lists`

各コレクションに対して以下のインデックスが作成されます：

1. **Embedding フィールドのベクトルインデックス**
   - ベクトル次元: 256
   - 構成: flat

2. **CreatedAt + Embedding 複合インデックス**
   - CreatedAt: 降順
   - Embedding: ベクトル（次元: 256）

3. **Status + CreatedAt 複合インデックス** (`tickets` コレクションのみ)
   - Status: 昇順
   - CreatedAt: 降順

## トラブルシューティング

### gcloud CLI の認証エラー

```bash
gcloud auth login
gcloud config set project your-project-id
```

### 権限エラー

Firestore の管理権限が必要です。以下のロールが必要です：
- `roles/datastore.indexAdmin`
- `roles/datastore.viewer` 