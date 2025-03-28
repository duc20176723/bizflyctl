/*
Copyright © (2020-2021) Bizfly Cloud

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/bizflycloud/bizflyctl/formatter"
	"github.com/bizflycloud/gobizfly"
)

var (
	kubernetesClusterHeader    = []string{"ID", "Name", "VPC Network ID", "Worker Pools Count", "Cluster Status", "Tags", "Created At", "Cluster Version"}
	detailKubernetesCluster    = []string{"ID", "Name", "VPC Network ID", "Worker Pools", "Worker Pools Count", "Cluster Status", "Tags", "Created At", "Cluster Version"}
	kubernetesWorkerPoolHeader = []string{
		"ID", "Name", "Version", "Flavor", "Volume Size", "Volume Type", "Nodes",
		"Enabled AutoScaling", "Min Size", "Max Size", "Created At",
	}
	clusterName              string
	clusterVersion           string
	vpcNetworkID             string
	tags                     []string
	workerPools              []string
	desiredSize              int
	enableAutoScaling        bool
	minSize                  int
	maxSize                  int
	outputKubeConfigFilePath string
	inputConfigFile          string
	expireTime               string
)

var kubernetesCmd = &cobra.Command{
	Use:   "kubernetes",
	Short: "Bizfly Kubernetes Engine Interaction",
	Long:  "Bizfly Kubernetes Engine Action: List, Create, Delete, Get",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help() // Display the help message
	},
}

var kubernetesWorkerPoolCmd = &cobra.Command{
	Use:   "workerpool",
	Short: "Bizfly Kubernetes Engine Worker Pool Interaction",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help() // Display the help message
	},
}

var kubernetesKubeConfigCmd = &cobra.Command{
	Use:   "kubeconfig",
	Short: "Bizfly Kubernetes Engine Kubeconfig Interaction",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help() // Display the help message
	},
}

var kubernetesNodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Bizfly Kubernetes Engine Node Interaction",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help() // Display the help message
	},
}

var clusterList = &cobra.Command{
	Use:   "list",
	Short: "List your Kubernetes cluster",
	Long:  "List your Kubernetes cluster",
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		clusters, err := client.KubernetesEngine.List(ctx, &gobizfly.ListOptions{})
		if err != nil {
			log.Fatal(err)
		}
		var data [][]string
		for _, cluster := range clusters {
			data = append(data, []string{
				cluster.UID, cluster.Name, cluster.VPCNetworkID, strconv.Itoa(cluster.WorkerPoolsCount),
				cluster.ClusterStatus, strings.Join(cluster.Tags, ", "), cluster.CreatedAt, cluster.Version.K8SVersion,
			})
		}
		formatter.Output(kubernetesClusterHeader, data)
	},
}

var clusterCreate = &cobra.Command{
	Use:   "create",
	Short: "Create Kubernetes cluster with worker pool",
	Long: `Create Kubernetes cluster with worker pool using file or flags (Sample config file in example)
- Using flag example: ./bizfly kubernetes create --name test_cli --version 5f7d3a91d857155ad4993a32 --vpc-network-id 145bed1f-a7f7-4f88-ab3d-ce2fc95a4e71 -tag abc -tag xyz --worker-pool "name=testworkerpool;flavor=nix.3c_6g;profile_type=premium;volume_type=PREMIUM-HDD1;volume_size=40;availability_zone=HN1;desired_size=1;min_size=1;max_size=10;labels=env=dev;taints=app=demo:NoSchedule"
- Using config file example: ./bizfly kubernetes create --config-file create_cluster.yml`,
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		var data [][]string
		if inputConfigFile != "" {
			fileBytes, err := os.ReadFile(inputConfigFile)
			if err != nil {
				log.Fatal(err)
			}
			var ccr *gobizfly.ClusterCreateRequest
			if err := yaml.Unmarshal(fileBytes, &ccr); err != nil {
				log.Fatal(err)
			}
			cluster, err := client.KubernetesEngine.Create(ctx, ccr)
			if err != nil {
				log.Fatal(err)
			}
			data = append(data, []string{
				cluster.UID, cluster.Name, cluster.VPCNetworkID, strconv.Itoa(cluster.WorkerPoolsCount),
				cluster.ClusterStatus, strings.Join(cluster.Tags, ", "), cluster.CreatedAt, cluster.Version.K8SVersion,
			})
			formatter.Output(kubernetesClusterHeader, data)
		} else {
			workerPoolObjs := make([]gobizfly.WorkerPool, 0)
			for _, pool := range workerPools {
				workerPoolObjs = append(workerPoolObjs, parseWorkerPool(pool))
			}
			cluster, err := client.KubernetesEngine.Create(ctx, &gobizfly.ClusterCreateRequest{
				Name:         clusterName,
				Version:      clusterVersion,
				VPCNetworkID: vpcNetworkID,
				WorkerPools:  workerPoolObjs,
				Tags:         tags,
			})
			if err != nil {
				log.Fatal(err)
			}
			data = append(data, []string{
				cluster.UID, cluster.Name, cluster.VPCNetworkID, strconv.Itoa(cluster.WorkerPoolsCount),
				cluster.ClusterStatus, strings.Join(cluster.Tags, ", "), cluster.CreatedAt, cluster.Version.K8SVersion,
			})
			formatter.Output(kubernetesClusterHeader, data)
		}
	},
}

var clusterGet = &cobra.Command{
	Use:   "get",
	Short: "Get Kubernetes cluster with worker pool",
	Long: `Get detail of cluster. 
- Using example: bizfly kubernetes get <cluster id>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("Invalid arguments")
			_ = cmd.Help() // Display the help message
			return
		}
		client, ctx := getApiClient(cmd)
		cluster, err := client.KubernetesEngine.Get(ctx, args[0])
		if err != nil {
			log.Fatal(err)
		}
		var data [][]string
		var workerPoolIds []string
		for _, workerPool := range cluster.WorkerPools {
			workerPoolIds = append(workerPoolIds, workerPool.UID)
		}
		data = append(data, []string{
			cluster.UID, cluster.Name, cluster.VPCNetworkID, strings.Join(workerPoolIds, "\n"), strconv.Itoa(cluster.WorkerPoolsCount),
			cluster.ClusterStatus, strings.Join(cluster.Tags, ", "), cluster.CreatedAt, cluster.Version.K8SVersion,
		})
		formatter.Output(detailKubernetesCluster, data)
	},
}

var clusterDelete = &cobra.Command{
	Use:   "delete",
	Short: "Delete Kubernetes cluster with worker pools",
	Long: `Delete a kubernetes cluster and all worker pools
- Using example: bizfly kubernetes delete <cluster id>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("Invalid arguments")
			_ = cmd.Help() // Display the help message
			return
		}
		client, ctx := getApiClient(cmd)
		err := client.KubernetesEngine.Delete(ctx, args[0])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Cluster is in the process of being deleted")
	},
}

var addWorkerPool = &cobra.Command{
	Use:   "add",
	Short: "Add worker pool into cluster",
	Long: `Add Kubernetes worker pool using file or flags (Sample config file in example)
- Using flag example: ./bizfly kubernetes workerpool add xfbxsws38dcs8o94 --worker-pool name=testworkerpool;flavor=nix.3c_6g;profile_type=premium;volume_type=PREMIUM-HDD1;volume_size=40;availability_zone=HN1;desired_size=1;min_size=1;max_size=10;labels=env=dev;taints=app=demo:NoSchedule
- Using config file example: ./bizfly kubernetes add-workerpool 55viixy9ma6yaiwu --config-file add_pools.yml`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("Invalid arguments")
			_ = cmd.Help() // Display the help message
			return
		}
		client, ctx := getApiClient(cmd)
		var data [][]string
		if inputConfigFile != "" {
			fileBytes, err := os.ReadFile(inputConfigFile)
			if err != nil {
				log.Fatal(err)
			}
			var awpr *gobizfly.AddWorkerPoolsRequest
			if err := yaml.Unmarshal(fileBytes, &awpr); err != nil {
				log.Fatal(err)
			}
			workerPools, err := client.KubernetesEngine.AddWorkerPools(ctx, args[0], awpr)
			if err != nil {
				log.Fatal(err)
			}
			for _, workerPool := range workerPools {
				data = append(data, []string{
					workerPool.UID, workerPool.Name, workerPool.Version, workerPool.Flavor,
					strconv.Itoa(workerPool.VolumeSize), workerPool.VolumeType, strconv.FormatBool(workerPool.EnableAutoScaling),
					strconv.Itoa(workerPool.MinSize), strconv.Itoa(workerPool.MaxSize), workerPool.CreatedAt,
				})
			}
			formatter.Output(kubernetesWorkerPoolHeader, data)

		} else {
			workerPoolObjs := make([]gobizfly.WorkerPool, 0)
			for _, pool := range workerPools {
				workerPoolObjs = append(workerPoolObjs, parseWorkerPool(pool))
			}
			workerPools, err := client.KubernetesEngine.AddWorkerPools(ctx, args[0], &gobizfly.AddWorkerPoolsRequest{
				WorkerPools: workerPoolObjs,
			})
			if err != nil {
				log.Fatal(err)
			}
			for _, workerPool := range workerPools {
				data = append(data, []string{
					workerPool.UID, workerPool.Name, workerPool.Version, workerPool.Flavor,
					strconv.Itoa(workerPool.VolumeSize), workerPool.VolumeType, strconv.FormatBool(workerPool.EnableAutoScaling),
					strconv.Itoa(workerPool.MinSize), strconv.Itoa(workerPool.MaxSize), workerPool.CreatedAt,
				})

				formatter.Output(kubernetesWorkerPoolHeader, data)
			}
		}
	},
}

var recycleNode = &cobra.Command{
	Use:   "recycle",
	Short: "Recycle Node",
	Long: `Recycle a node in a worker pool in a cluster 
Using example: bizfly kubernetes workerpool node recycle <cluster id> <workerpool id> <node id>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 3 {
			fmt.Println("Invalid arguments")
			_ = cmd.Help() // Display the help message
			return
		}
		client, ctx := getApiClient(cmd)
		err := client.KubernetesEngine.RecycleNode(ctx, args[0], args[1], args[2])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Recycling node successfully")
	},
}

var deleteWorkerPool = &cobra.Command{
	Use:   "delete",
	Short: "Delete worker pool",
	Long: `Delete a worker pool in a kubernetes cluster
- Using example: bizfly kubernetes workerpool delete <cluster id> <worker pool id>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			fmt.Println("Invalid arguments")
			_ = cmd.Help() // Display the help message
			return
		}
		client, ctx := getApiClient(cmd)
		err := client.KubernetesEngine.DeleteClusterWorkerPool(ctx, args[0], args[1])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Worker pool is deleting now")
	},
}

var getWorkerPool = &cobra.Command{
	Use:   "get",
	Short: "Get worker pool",
	Long: `Get detail of worker pool in a kubernetes cluster
- Using example: bizfly kubernetes workerpool get <cluster id> <worker pool id>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			fmt.Println("Invalid arguments")
			_ = cmd.Help() // Display the help message
			return
		}
		client, ctx := getApiClient(cmd)
		workerPool, err := client.KubernetesEngine.GetClusterWorkerPool(ctx, args[0], args[1])
		if err != nil {
			log.Fatal(err)
		}
		var data [][]string
		var nodes []string
		for _, node := range workerPool.Nodes {
			nodes = append(nodes, node.PhysicalID)
		}
		data = append(data, []string{
			workerPool.UID, workerPool.Name, workerPool.Version, workerPool.Flavor,
			strconv.Itoa(workerPool.VolumeSize), workerPool.VolumeType, strings.Join(nodes, "\n"), strconv.FormatBool(workerPool.EnableAutoScaling),
			strconv.Itoa(workerPool.MinSize), strconv.Itoa(workerPool.MaxSize), workerPool.CreatedAt,
		})
		formatter.Output(kubernetesWorkerPoolHeader, data)
	},
}

var updateWorkerPool = &cobra.Command{
	Use:   "update",
	Short: "Update worker pool",
	Long: `Update a worker pool in a cluster
- Using example: bizfly kubnernetes worker pool update <cluster id> <workerpool id> --desired-size <size> --min-size <size> --max-size <size> --autoscaling <true|false>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			fmt.Println("Invalid arguments")
			_ = cmd.Help() // Display the help message
			return
		}

		client, ctx := getApiClient(cmd)
		uwr := &gobizfly.UpdateWorkerPoolRequest{
			DesiredSize:       desiredSize,
			EnableAutoScaling: enableAutoScaling,
			MinSize:           minSize,
			MaxSize:           maxSize,
		}
		err := client.KubernetesEngine.UpdateClusterWorkerPool(ctx, args[0], args[1], uwr)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Worker pool is updating now")
	},
}

var deleteWorkerPoolNode = &cobra.Command{
	Use:   "delete",
	Short: "Delete node",
	Long: `Delete a node in a worker pool in a cluster 
Using example: bizfly kubernetes workerpool node delete <cluster id> <worker pool id> <node id>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 3 {
			fmt.Println("Invalid arguments")
			_ = cmd.Help() // Display the help message
			return
		}
		client, ctx := getApiClient(cmd)
		err := client.KubernetesEngine.DeleteClusterWorkerPoolNode(ctx, args[0], args[1], args[2])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Worker pool is in the process of being deleted")
	},
}

var getKubeConfig = &cobra.Command{
	Use:   "get",
	Short: "Get kubeconfig",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("Invalid arguments")
			_ = cmd.Help() // Display the help message
			return
		}
		client, ctx := getApiClient(cmd)
		kubeconfigOptions := &gobizfly.GetKubeConfigOptions{
			ExpiteTime: expireTime,
		}
		resp, err := client.KubernetesEngine.GetKubeConfig(ctx, args[0], kubeconfigOptions)
		if err != nil {
			log.Fatal(err)
		}

		currentDir, _ := os.Getwd()

		defaultFileName := fmt.Sprintf("%s.kubeconfig", args[0])

		stat, err := os.Stat(outputKubeConfigFilePath)
		if err == nil && stat.IsDir() {
			outputKubeConfigFilePath = filepath.Join(outputKubeConfigFilePath, defaultFileName)
		} else if !filepath.IsAbs(outputKubeConfigFilePath) {
			// input path is relative file path
			outputKubeConfigFilePath = filepath.Join(currentDir, outputKubeConfigFilePath)
		}

		file, _ := os.Create(outputKubeConfigFilePath)
		_, err = file.WriteString(resp)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Get kubernetes config successfully. Output path:", outputKubeConfigFilePath)
	},
}

func isIntField(key string) bool {
	for _, field := range []string{"volume_size", "desired_size", "min_size", "max_size"} {
		if field == key {
			return true
		}
	}
	return false
}

func parseTaints(pair string) []gobizfly.Taint {
	r := regexp.MustCompile("(.*)=(.*):(.*)")
	rTaints := regexp.MustCompile(`taints=(.*)`)
	subStrs := rTaints.FindStringSubmatch(pair)
	if len(subStrs) == 0 {
		log.Fatal("Invalid worker pool taints input")
	}
	values := subStrs[1]
	taintPairs := strings.Split(values, ",")
	taints := make([]gobizfly.Taint, 0)
	for _, taintPair := range taintPairs {
		subStrs := r.FindStringSubmatch(taintPair)
		fmt.Println(subStrs)
		if len(subStrs) == 0 {
			log.Fatal("Invalid worker pool taints input")
		}
		if subStrs[3] == "" || subStrs[1] == "" {
			log.Fatal("Invalid worker pool taints input")
		}
		taint := gobizfly.Taint{
			Effect: subStrs[3],
			Key:    subStrs[1],
			Value:  subStrs[2],
		}
		taints = append(taints, taint)
	}
	return taints
}

func parseLabels(pair string) map[string]string {
	r := regexp.MustCompile("(.*)=(.*)")
	rLabels := regexp.MustCompile(`labels=(.*)`)
	subStrs := rLabels.FindStringSubmatch(pair)
	if len(subStrs) == 0 {
		log.Fatal("Invalid worker pool labels input")
	}
	values := subStrs[1]
	labelPairs := strings.Split(values, ",")
	labelsMap := make(map[string]string)
	for _, labelPair := range labelPairs {
		subStrs := r.FindStringSubmatch(labelPair)
		if len(subStrs) == 0 {
			log.Fatal("Invalid worker pool labels input")
		}
		labelsMap[subStrs[1]] = subStrs[2]
	}
	return labelsMap
}

func parseWorkerPool(workerPoolStr string) gobizfly.WorkerPool {
	pairs := strings.Split(workerPoolStr, ";")
	strRequiredFields := []string{"name", "flavor", "profile_type", "volume_type", "availability_zone"}
	intRequiredFields := []string{"volume_size", "desired_size", "min_size", "max_size"}
	strFieldMap := make(map[string]string)
	intFieldMap := make(map[string]int)
	mapFieldMap := make(map[string]map[string]string)
	taintsField := make([]gobizfly.Taint, 0)
	isEnableAutoScaling := false
	r := regexp.MustCompile("(.*)=(.*)")
	for _, pair := range pairs {
		if strings.Contains(pair, "labels") {
			mapFieldMap["labels"] = parseLabels(pair)
			continue
		}
		if strings.Contains(pair, "taints") {
			taintsField = parseTaints(pair)
			continue
		}
		subStrs := r.FindStringSubmatch(pair)
		if len(subStrs) == 0 {
			log.Fatal("Invalid worker pool input")
		}
		fmt.Println(subStrs, len(subStrs))
		key, value := subStrs[1], subStrs[2]
		if key == "enable_autoscaling" {
			b, _ := strconv.ParseBool(value)
			isEnableAutoScaling = b
		}
		if isIntField(key) {
			i, _ := strconv.Atoi(value)
			intFieldMap[key] = i
		} else {
			strFieldMap[key] = value
		}
	}
	for _, field := range strRequiredFields {
		if strFieldMap[field] == "" {
			log.Fatal("Missing required worker pool field: ", field)
		}
	}
	for _, field := range intRequiredFields {
		if intFieldMap[field] == 0 {
			log.Fatal("Missing required worker pool field: ", field)
		}
	}
	workerPool := gobizfly.WorkerPool{
		Name:              strFieldMap["name"],
		Flavor:            strFieldMap["flavor"],
		ProfileType:       strFieldMap["profile_type"],
		VolumeType:        strFieldMap["volume_type"],
		VolumeSize:        intFieldMap["volume_size"],
		AvailabilityZone:  strFieldMap["availability_zone"],
		DesiredSize:       intFieldMap["desired_size"],
		EnableAutoScaling: isEnableAutoScaling,
		MinSize:           intFieldMap["min_size"],
		MaxSize:           intFieldMap["max_size"],
		Labels:            mapFieldMap["labels"],
		Taints:            taintsField,
	}
	fmt.Printf("WorkerPool %v+", workerPool)
	return workerPool
}

func init() {
	rootCmd.AddCommand(kubernetesCmd)
	kubernetesCmd.AddCommand(kubernetesWorkerPoolCmd)
	kubernetesWorkerPoolCmd.AddCommand(kubernetesNodeCmd)
	kubernetesCmd.AddCommand(kubernetesKubeConfigCmd)

	kubernetesCmd.AddCommand(clusterList)
	kubernetesCmd.AddCommand(clusterDelete)
	kubernetesCmd.AddCommand(clusterGet)
	kubernetesWorkerPoolCmd.AddCommand(deleteWorkerPool)
	kubernetesWorkerPoolCmd.AddCommand(getWorkerPool)

	kubernetesNodeCmd.AddCommand(deleteWorkerPoolNode)
	kubernetesNodeCmd.AddCommand(recycleNode)

	kccq := clusterCreate.Flags()
	kccq.StringVar(&inputConfigFile, "config-file", "", "Input config file")
	kccq.StringVar(&clusterName, "name", "", "Name of cluster")
	kccq.StringVar(&clusterVersion, "version", "", "Version of cluster")
	kccq.StringVar(&vpcNetworkID, "vpc-network-id", "", "VPC Network ID")
	kccq.StringArrayVar(&tags, "tag", []string{}, "Tags of cluster")
	kccq.StringArrayVar(&workerPools, "worker-pool", []string{}, "Worker pools")
	_ = clusterCreate.MarkFlagRequired("name")
	_ = clusterCreate.MarkFlagRequired("version")
	_ = clusterCreate.MarkFlagRequired("vpc-network-id")
	_ = clusterCreate.MarkFlagRequired("worker-pool")
	kubernetesCmd.AddCommand(clusterCreate)

	awp := addWorkerPool.PersistentFlags()
	awp.StringVar(&inputConfigFile, "config-file", "", "Input config file")
	awp.StringArrayVar(&workerPools, "worker-pool", []string{}, "Worker pools")
	kubernetesWorkerPoolCmd.AddCommand(addWorkerPool)

	uwp := updateWorkerPool.Flags()
	uwp.IntVar(&desiredSize, "desired-size", 1, "Desired size")
	uwp.BoolVar(&enableAutoScaling, "autoscaling", false, "Enable Auto scaling")
	uwp.IntVar(&minSize, "min-size", 1, "Min size")
	uwp.IntVar(&maxSize, "max-size", 1, "Max size")

	_ = updateWorkerPool.MarkFlagRequired("desired-size")
	_ = updateWorkerPool.MarkFlagRequired("min-size")
	_ = updateWorkerPool.MarkFlagRequired("max-size")

	kubernetesWorkerPoolCmd.AddCommand(updateWorkerPool)

	getKubeConfig.PersistentFlags().StringVar(&outputKubeConfigFilePath, "output", ".", "Output path")
	getKubeConfig.PersistentFlags().StringVar(&expireTime, "expire-time", "3000", "Set kubeconfig's expire time")
	kubernetesKubeConfigCmd.AddCommand(getKubeConfig)
}
