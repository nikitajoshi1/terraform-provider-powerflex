/*
Copyright (c) 2023 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Mozilla Public License Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://mozilla.org/MPL/2.0/


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

# commands to run this tf file : terraform init && terraform apply --auto-approve
# This datasource reads volumes either by id or name or storage_pool_id or storage_pool_name where user can provide a value to any one of them
# If it is a empty datsource block , then it will read all the volumes
# If id or name is provided then it reads a particular volume with that id or name
# If storage_pool_id or storage_pool_name is provided then it will return the volumes under that storage pool
# Only one of the attribute can be provided among id, name, storage_pool_id, storage_pool_name 

data "powerflex_volume" "volume" {

  #name = "cosu-ce5b8a2c48"
  id = "4570761d00000024"
  #storage_pool_id= "c98e26e500000000"
  #storage_pool_name= "pool2"
}

output "volumeResult" {
  value = data.powerflex_volume.volume.volumes
}

