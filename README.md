# Pocket

Pocket borrows [pingcap/chaos](https://github.com/pingcap/chaos)'s way to manage workloads of TiDB cluster running on K8s, and uses [pingcap/chaos-mesh](https://github.com/pingcap/chaos-mesh) to perform nemesises.

# Requirements

* [TiDB Operator](https://github.com/pingcap/tidb-operator) >= v1.1.0-beta.1

* [Chaos Mesh](https://github.com/pingcap/chaos-mesh)

# Workloads

* bank

* multibank

* longfork

# Chaos

* Pod failure(random/minor/major/all)

* Network partition

