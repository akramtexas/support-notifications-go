/*******************************************************************************
 * Copyright 2017 Dell Inc.
 * Copyright 2018 Dell Technologies Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *
 * @microservice: support-notifications
 * @author: Jim White, Dell Technologies
 * @version: 0.5.0
 *******************************************************************************/
package main

import (
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/edgexfoundry/edgex-go"
	"github.com/edgexfoundry/edgex-go/pkg/config"
	"github.com/edgexfoundry/edgex-go/pkg/heartbeat"
	"github.com/edgexfoundry/edgex-go/support/logging-client"
	"github.com/edgexfoundry/edgex-go/support/notifications"
)

var loggingClient logger.LoggingClient

func main() {
	start := time.Now()
	var (
		useConsul  = flag.String("consul", "", "Should the service use consul?")
		useProfile = flag.String("profile", "default", "Specify a profile other than default.")
	)
	flag.Parse()

	//Read Configuration
	configuration := &notifications.ConfigurationStruct{}
	err := config.LoadFromFile(*useProfile, configuration)
	if err != nil {
		logBeforeTermination(err)
		return
	}

	//Determine if configuration should be overridden from Consul
	var consulMsg string
	if *useConsul == "y" {
		consulMsg = "Loading configuration from Consul..."
		err := notifications.ConnectToConsul(*configuration)
		if err != nil {
			logBeforeTermination(err)
			return //end program since user explicitly told us to use Consul.
		}
	} else {
		consulMsg = "Bypassing Consul configuration..."
	}

	// Setup Logging
	logTarget := setLoggingTarget(*configuration)
	loggingClient = logger.NewClient(configuration.ApplicationName, configuration.EnableRemoteLogging, logTarget)

	loggingClient.Info(consulMsg)
	loggingClient.Info(fmt.Sprintf("Starting %s %s ", notifications.SUPPORTNOTIFICATIONSSERVICENAME, edgex.Version))

	err = notifications.Init(*configuration, loggingClient)
	if err != nil {
		loggingClient.Error(fmt.Sprintf("call to init() failed: %v", err.Error()))
		return
	}

	r := notifications.LoadRestRoutes()
	http.TimeoutHandler(nil, time.Millisecond*time.Duration(configuration.ServiceTimeout), "Request timed out")
	loggingClient.Info(configuration.AppOpenMsg, "")

	heartbeat.Start(configuration.HeartBeatMsg, configuration.HeartBeatTime, loggingClient)

	// Time it took to start service
	loggingClient.Info("Service started in: "+time.Since(start).String(), "")
	loggingClient.Info("Listening on port: " + strconv.Itoa(configuration.ServicePort))
	loggingClient.Error(http.ListenAndServe(":"+strconv.Itoa(configuration.ServicePort), r).Error())
}

func logBeforeTermination(err error) {
	loggingClient = logger.NewClient(notifications.SUPPORTNOTIFICATIONSSERVICENAME, false, "")
	loggingClient.Error(err.Error())
}

func setLoggingTarget(conf notifications.ConfigurationStruct) string {
	logTarget := conf.LoggingRemoteURL
	if !conf.EnableRemoteLogging {
		return conf.LoggingFile
	}
	return logTarget
}
