package grafana

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFolder_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var folder gapi.Folder
	var folderWithUID gapi.Folder

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccFolderCheckDestroy(&folder),
			testAccFolderCheckDestroy(&folderWithUID),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_folder/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "id", idRegexp),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "uid", uidRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder", "title", "Terraform Test Folder"),

					testAccFolderCheckExists("grafana_folder.test_folder_with_uid", &folderWithUID),
					resource.TestMatchResourceAttr("grafana_folder.test_folder_with_uid", "id", idRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "uid", "test-folder-uid"),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "title", "Terraform Test Folder With UID"),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/dashboards/f/test-folder-uid/terraform-test-folder-with-uid"),
				),
			},
			{
				ResourceName:      "grafana_folder.test_folder",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_folder.test_folder_with_uid",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccFolderCheckExists(rn string, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}

		if id == 0 {
			return fmt.Errorf("got a folder id of 0")
		}
		gotFolder, err := client.Folder(id)
		if err != nil {
			return fmt.Errorf("error getting folder: %s", err)
		}

		*folder = *gotFolder

		return nil
	}
}

func testAccFolderCheckDestroy(folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		_, err := client.Folder(folder.ID)
		if err == nil {
			return fmt.Errorf("folder still exists")
		}
		return nil
	}
}
