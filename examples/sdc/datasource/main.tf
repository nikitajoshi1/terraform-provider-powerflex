# cd ../../.. && make install && cd examples/sdc/datasource
# terraform init && terraform apply --auto-approve
terraform {
  required_providers {
    powerflex = {
      version = "0.1"
      source  = "dell.com/dev/powerflex"
    }
  }
}

provider "powerflex" {
    username = ""
    password = ""
    host = ""
    insecure = ""
    usecerts = ""
    powerflex_version = ""
}

# # -----------------------------------------------------------------------------------
# # Read all sdcs if id is blank, otherwise reads all sdcs
# # -----------------------------------------------------------------------------------
    # systemid is required field
    # name is optional if empty then will return all sdc
    # sdcid is optional if empty then will return all sdc
    # sdcid and name both are empty then will return all sdc
data "powerflex_sdc" "selected" {
    # sdcid = "595a0bb100000001"
    systemid = "0e7a082862fedf0f"
    name = "LGLW6090" // LGLW6091
}

# # Returns all sdcs matching criteria
output "allsdcresult" {
  value = data.powerflex_sdc.selected.sdcs
}
# # -----------------------------------------------------------------------------------

