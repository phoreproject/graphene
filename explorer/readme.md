## Environment

* Postgres
* NodeJs

## Modules

### Synapse Go backend

#### Build

The Go backend requires all 4 components: Beacon, Shard, Validator, and Explorer.

#### Run

Below are example commands to run the 4 component

```
synapsebeacon -level trace -listen /ip4/0.0.0.0/tcp/11781 -rpclisten /ip4/127.0.0.1/tcp/11782 -chaincfg testnet.json -datadir=DATADIR

synapseshard.exe -level trace -beacon /ip4/127.0.0.1/tcp/11782 -listen /ip4/127.0.0.1/tcp/11783

synapsevalidator -beacon /ip4/127.0.0.1/tcp/11782 -shard /ip4/127.0.0.1/tcp/11783 -rootkey testnet -networkid testnet -validators 0-255

synapseexplorer -dbdriver postgres -dbhost localhost -dbdatabase synapse -dbuser test -dbpassword "test" -level trace -listen /ip4/0.0.0.0/tcp/21781 -rpclisten /ip4/0.0.0.0/tcp/11782 -shardlisten /ip4/127.0.0.1/tcp/11783  -chainconfig regtest.temp.json -datadir=DATADIR -connect /ip4/0.0.0.0/tcp/11781/ipfs/BEACON_IPFS_ADDRESS
```

### Website backend

The website backend is in folder explorer/website/backend

#### Install dependencies

Go to the backend folder, run,

`npm install`

#### Config the backend

Go to backend_folder/config, edit default.json. The file is self documented.

#### Run the backend (web server)

Go to the backend folder, run,

`node index.js`

### Website frontend

The website backend is in folder explorer/website/frontend

#### Install dependencies

Go to the frontend folder, run,

`npm install`

#### Build the frontend

Go to the frontend folder, run,

`npm run prod`

To build development version, run,

`npm run dev`

