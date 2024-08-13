package gpumemoryfilter

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const Name = "GpuMemoryFilter"

var _ = framework.FilterPlugin(&CustomFilter{})

// CustomFilter is a plugin that filters out nodes based on custom logic.
type CustomFilter struct{}
type Response struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}
type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}
type Result struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"` // interface{} to handle mixed types
}

// Name returns the name of the plugin.
func (pl *CustomFilter) Name() string {
	return Name
}

// Filter is the method to implement the filtering logic.
func (pl *CustomFilter) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	// Example filtering logic:
	// Disallow scheduling on nodes with a specific label.
	if !findBestNodeForPod(nodeInfo.Node().Name) {
		return framework.NewStatus(framework.Unschedulable, "Node has disallow-schedule label")
	}

	return nil // nil means the node passed the filter
}

func findBestNodeForPod(nodeName string) bool {

	url := "http://localhost:8080/api/v1/query?query=DCGM_FI_DEV_FB_USED{kubernetes_node%3D%22" + nodeName + "%22}"
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching URL:", err)
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return false
	}
	var response Response
	err = json.Unmarshal(body, &response)
	if len(response.Data.Result) > 0 && len(response.Data.Result[0].Value) > 1 {
		// Convert the second value in the Value array to float64
		valueStr, ok := response.Data.Result[0].Value[1].(string)
		if !ok {
			fmt.Println("Value is not a string, cannot convert to float.")
		}
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			fmt.Println("Error converting value to float:", err)
		}
		fmt.Printf("Node %s GPU Usage: %.2f\n", nodeName, value)
		// Check if the value is greater than 2000
		if value < 22000 {
			fmt.Printf("Node %s has GPU usage over 2000!\n", nodeName)
			fmt.Println("Node gpu", response.Data.Result[0].Value[1])
			return true
		} else {
			fmt.Printf("Node %s GPU usage is under the limit.\n", nodeName)
		}
	} else {
		fmt.Println("No results or insufficient data in results.")
	}
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return false
	}
	return false
}

// New creates a new CustomFilter plugin.
func New(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &CustomFilter{}, nil
}
