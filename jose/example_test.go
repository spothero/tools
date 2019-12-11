// Copyright 2019 SpotHero
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jose

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	jwt := ""
	c := Config{}
	cmd := &cobra.Command{
		Use:   "jose",
		Short: "jose test app",
		Run: func(cmd *cobra.Command, args []string) {
			test(c, jwt)
		},
	}
	flags := cmd.Flags()
	c.RegisterFlags(flags)
	flags.StringVar(&jwt, "jwt", "", "The JWT to parse")
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func test(c Config, jwt string) {
	if len(jwt) == 0 {
		fmt.Printf("failed to supply jwt")
		os.Exit(1)
	}
	c.ClaimGenerators = []ClaimGenerator{CognitoGenerator{}}
	client, err := c.NewJOSE()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	claims := client.GetClaims()
	if err := client.ParseValidateJWT(jwt, claims...); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse token: %+v\n", err)
		os.Exit(1)
	}
	fmt.Printf("successfully parsed token: %+v\n", claims[0])
}
