package migrations

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

var __20200319122755_create_repositories_table_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\x28\x4a\x2d\xc8\x2f\xce\x2c\xc9\x2f\xca\x4c\x2d\xb6\x06\x04\x00\x00\xff\xff\x3c\x53\x72\x9a\x22\x00\x00\x00")

func _20200319122755_create_repositories_table_down_sql() ([]byte, error) {
	return bindata_read(
		__20200319122755_create_repositories_table_down_sql,
		"20200319122755_create_repositories_table.down.sql",
	)
}

var __20200319122755_create_repositories_table_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x94\x91\x41\x6f\x9b\x30\x18\x86\xef\xfc\x8a\xf7\x08\xd2\x4e\x93\x72\xda\x76\x70\xe0\x23\xb3\xc2\xcc\x66\x8c\x34\x4e\x88\x05\x2f\x58\x6d\x80\x12\x47\xa9\xfa\xeb\x2b\x20\x24\x4d\xa8\x1a\xd5\x37\x8c\xdf\xe7\x7d\xfc\xd9\x97\xc4\x14\x41\xb1\x65\x44\xe0\x21\x44\xac\x40\x7f\x79\xa2\x12\x74\xba\x6d\xf6\xc6\x36\x9d\xd1\x7b\xc7\x75\x00\xc0\x94\x98\xd6\x3f\xb3\x35\xb5\xc5\x7c\xf5\x04\x91\x46\x11\x56\x24\x48\x32\x45\x01\x96\x19\x02\x0a\x59\x1a\x29\xb0\x04\x3c\x20\xa1\xb8\xca\xbe\x0c\xc8\xb6\xe8\x74\x6d\xf3\x9e\x3c\x22\xc7\xed\x4d\xa7\x0b\xab\xcb\xbc\xb0\xb0\x66\xa7\xf7\xb6\xd8\xb5\x38\x1a\x5b\x0d\x9f\x78\x69\x6a\x7d\x69\x9a\xe0\x75\x73\x74\xbd\x31\x7f\x68\xcb\x7b\xf9\xf1\x60\x5d\xec\xf4\xc9\xdc\xea\xe7\xf7\x2e\x74\xb9\xd2\x64\x6c\xab\xcf\x25\xfc\x58\x24\x4a\x32\x2e\x14\xda\x87\xfc\xed\x5c\xf1\x5b\xf2\x5f\x4c\x66\x58\x53\x06\xd7\x94\xde\x2c\xf0\xff\x3a\x90\x9f\xe7\x75\xcd\x09\x63\x49\x7c\x25\x46\xce\xf9\x8c\xe7\x4c\x42\x92\x42\x92\x24\x7c\xba\x7e\xd7\xa1\x13\xb1\x40\x40\x11\x29\x82\xcf\x12\x9f\x05\x34\xb3\x38\x3c\xdd\x5a\xd8\x0a\xa9\xe0\x7f\x52\xea\xeb\x6c\x35\x17\xdf\xdc\x88\xf7\x83\xce\x1f\x75\xbd\xb5\x15\xfc\x9f\xe4\xaf\xe1\xba\x9b\xaa\xe8\x4e\x7b\x6e\xff\xdf\xc3\xf7\x1f\xf8\xba\x58\x78\xf7\x71\x7d\xe9\x47\xb8\x41\xea\x8c\x73\xbc\x6f\xaf\x01\x00\x00\xff\xff\xa7\x1d\x45\x3c\xea\x02\x00\x00")

func _20200319122755_create_repositories_table_up_sql() ([]byte, error) {
	return bindata_read(
		__20200319122755_create_repositories_table_up_sql,
		"20200319122755_create_repositories_table.up.sql",
	)
}

var __20200319130108_create_manifests_table_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\xc8\x4d\xcc\xcb\x4c\x4b\x2d\x2e\x29\xb6\x06\x04\x00\x00\xff\xff\x22\x39\x5a\x0e\x1f\x00\x00\x00")

func _20200319130108_create_manifests_table_down_sql() ([]byte, error) {
	return bindata_read(
		__20200319130108_create_manifests_table_down_sql,
		"20200319130108_create_manifests_table.down.sql",
	)
}

var __20200319130108_create_manifests_table_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x91\x5f\x4f\x83\x30\x14\xc5\xdf\xf7\x29\xce\x23\x24\x3e\x99\xec\x49\x7d\x60\xac\x53\x32\xec\x14\x4a\x22\x4f\x4d\x85\x2b\x34\x1b\x05\xa1\xba\xcd\x4f\x6f\x1c\x3a\x96\xfd\x49\xbc\x6f\xbd\x39\xe7\xd7\x9e\x53\x3f\x62\x9e\x60\x10\xde\x24\x64\x08\x66\xe0\x0b\x01\xf6\x12\xc4\x22\x46\xa5\x8c\x7e\xa3\xce\x76\x23\x67\x04\x00\x3a\xc7\xe1\xbc\xea\x42\x1b\x8b\xd3\xf9\x41\xf0\x24\x0c\x71\xcf\x38\x8b\x3c\xc1\xa6\x98\xa4\x98\xb2\x99\x97\x84\x02\x5e\x8c\x60\xca\xb8\x08\x44\x7a\xb5\xc3\x66\x2d\x29\x4b\xb9\x54\x3d\xcb\xea\x8a\x3a\xab\xaa\x06\x6b\x6d\xcb\xdd\x11\x5f\xb5\xa1\x01\xfb\x47\x32\xf5\xda\x71\x7b\x46\xa5\xda\xe5\x80\xb8\xc8\xe8\xc5\x5d\x56\x52\xa5\xe4\x27\xb5\x9d\xae\x0d\xb4\xb1\x54\x50\x7b\x39\x47\xef\xca\x75\x41\x9d\x95\x25\x6d\xfa\xf4\x5b\x4b\xea\x4c\xf8\x23\x57\xa3\xb6\xab\x5a\xed\x8b\xfb\xa7\xab\xa2\x5c\x2b\x69\xb7\x0d\xf5\x71\x68\x73\xae\xe7\x63\x97\xbf\xe0\xb1\x88\xbc\x80\x0b\x34\x4b\xb9\xff\x3d\x3c\x45\xc1\xa3\x17\xa5\x98\xb3\x14\x8e\xce\xdd\x13\xf5\xc7\xfb\xa0\x96\x07\x39\x13\x1e\x3c\x27\x0c\xce\xb0\x3a\xf5\x66\x07\x37\xc9\xe1\xdd\x72\x45\xa6\xb0\x25\xfc\x07\xe6\xcf\xe1\x38\x59\xa9\xda\xdf\x9d\x33\xa8\x5c\xdc\xde\xe1\x7a\x3c\x76\xdd\x91\x7b\xf3\x1d\x00\x00\xff\xff\x4d\xe1\x25\x51\x8a\x02\x00\x00")

func _20200319130108_create_manifests_table_up_sql() ([]byte, error) {
	return bindata_read(
		__20200319130108_create_manifests_table_up_sql,
		"20200319130108_create_manifests_table.up.sql",
	)
}

var __20200319131222_create_manifest_configurations_table_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\xc8\x4d\xcc\xcb\x4c\x4b\x2d\x2e\x89\x4f\xce\xcf\x4b\xcb\x4c\x2f\x2d\x4a\x2c\xc9\xcc\xcf\x2b\xb6\x06\x04\x00\x00\xff\xff\xd0\x67\xff\x10\x2d\x00\x00\x00")

func _20200319131222_create_manifest_configurations_table_down_sql() ([]byte, error) {
	return bindata_read(
		__20200319131222_create_manifest_configurations_table_down_sql,
		"20200319131222_create_manifest_configurations_table.down.sql",
	)
}

var __20200319131222_create_manifest_configurations_table_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x92\xcf\x6e\x9c\x30\x18\xc4\xef\xfb\x14\x73\x04\xa9\xa7\x4a\x39\xb5\x3d\x38\xf0\x6d\x6a\x85\x9a\xd6\x18\xa9\x9c\x90\x03\x5e\xb0\x1a\xfe\x74\x71\x94\x6c\x9e\xbe\xca\xd2\x05\x14\xb2\xc9\xfa\x04\x96\xe7\x37\xf3\x79\x1c\x48\x62\x8a\xa0\xd8\x75\x44\xe0\x5b\x88\x58\x81\x7e\xf3\x44\x25\x68\x74\x6b\x77\x66\x70\x79\xd1\xb5\x3b\x5b\x3d\xec\xb5\xb3\x5d\x3b\x6c\xbc\x0d\x00\xd8\x12\xd3\xba\xb3\x95\x6d\x1d\xd6\xeb\x85\x26\xd2\x28\xc2\x0d\x09\x92\x4c\x51\x88\xeb\x0c\x21\x6d\x59\x1a\x29\xb0\x04\x3c\x24\xa1\xb8\xca\x3e\x1d\x99\x93\xa3\x2d\x3f\x66\x8e\x92\xc1\x3e\x9b\x8b\x63\x8c\x92\x62\x6f\xb4\x33\x65\xae\x1d\xe0\x6c\x63\x06\xa7\x9b\x1e\x8f\xd6\xd5\xc7\x5f\x3c\x77\xad\x99\x93\x9f\xc2\xb6\xdd\xa3\xe7\x8f\x80\xd2\x56\x2f\x21\x6b\xf3\x04\xdc\x1d\x9c\xd1\x6f\x58\xbe\xf2\xec\xf5\xe1\xbe\xd3\xe3\x95\x5d\x28\x69\x4c\x69\x75\xee\x0e\xbd\x01\x9c\x79\x7a\x6b\xae\xd7\x92\x20\x16\x89\x92\x8c\x0b\x85\xfe\x4f\x7e\xa6\x3f\xfc\x94\xfc\x07\x93\x19\x6e\x29\x83\x67\x4b\x7f\xa5\xdd\x9d\xd5\xe6\x8b\x86\xa6\xef\x01\xdb\x58\x12\xbf\x11\x23\x71\x71\xc4\xdf\x9c\x72\x4a\xda\x92\x24\x11\xd0\xfc\xac\x86\xa3\x39\x62\x81\x90\x22\x52\x84\x80\x25\x01\x0b\x69\x15\xe7\xe1\xef\xd9\x38\x8b\x26\x52\xc1\x7f\xa5\x04\x6f\xde\x5a\x0f\xf6\x0e\x69\xf9\xf4\x4e\xa8\xe5\x24\x2b\x56\xf1\xce\x25\x4d\xcd\xe5\xf7\xa6\xad\x5c\x8d\xe0\x3b\x05\xb7\xf0\xbc\xa2\xd6\xfb\xff\x7b\xde\x7c\xca\xc7\xd7\x6f\xf8\x7c\x75\xe5\xfb\x1b\xff\xcb\xbf\x00\x00\x00\xff\xff\x64\x60\x8f\x03\x91\x03\x00\x00")

func _20200319131222_create_manifest_configurations_table_up_sql() ([]byte, error) {
	return bindata_read(
		__20200319131222_create_manifest_configurations_table_up_sql,
		"20200319131222_create_manifest_configurations_table.up.sql",
	)
}

var __20200319131542_create_layers_table_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\xc8\x49\xac\x4c\x2d\x2a\xb6\x06\x04\x00\x00\xff\xff\x04\xc2\x07\x1b\x1c\x00\x00\x00")

func _20200319131542_create_layers_table_down_sql() ([]byte, error) {
	return bindata_read(
		__20200319131542_create_layers_table_down_sql,
		"20200319131542_create_layers_table.down.sql",
	)
}

var __20200319131542_create_layers_table_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x90\x3d\x6f\xc2\x30\x10\x86\x77\x7e\xc5\x3b\x3a\x52\xa7\x4a\x4c\x6d\x07\x13\x4c\x6b\x91\x9a\x36\x71\xa4\x66\xb2\x0c\x39\x11\x0b\x12\x68\xe2\x8a\x8f\x5f\x5f\x95\x80\x52\x11\x2a\xf5\x36\x9f\xde\xe7\xee\xf1\x85\xb1\xe0\x5a\x40\xf3\x51\x24\x20\x27\x50\x33\x0d\xf1\x21\x13\x9d\x60\x6d\x0f\x54\x37\x03\x36\x00\x00\x97\xe3\x52\x73\xb7\x74\x95\x47\xbf\x7e\x58\x95\x46\x11\x9e\x85\x12\x31\xd7\x62\x8c\x51\x86\xb1\x98\xf0\x34\xd2\xe0\x09\xe4\x58\x28\x2d\x75\x76\x77\x1a\xd9\xb8\x23\xfd\x77\x64\x4b\x2c\x6a\xb2\x9e\x72\x63\x3d\xbc\x2b\xa9\xf1\xb6\xdc\x62\xe7\x7c\x71\x7a\xe2\xb8\xa9\xa8\x93\xb8\xec\xad\x36\x3b\x16\xb4\x7c\x69\xeb\x55\x8b\xff\xc9\xb7\xc1\xdc\x2d\xa9\xf1\xa6\xa0\x3d\xe6\x07\x4f\xf6\x86\xd9\x95\x5a\x49\xb9\xb3\xc6\x1f\xb6\x04\x4f\xfb\x5b\x5f\xb9\x26\xc2\x99\x4a\x74\xcc\xa5\xd2\xd8\xae\x4c\x7b\x6c\xbc\xc5\xf2\x95\xc7\x19\xa6\x22\x03\x73\x79\xd0\x8b\x7e\x7d\x9e\xa3\xe6\x97\x63\xaa\xe4\x7b\x2a\xc0\xba\x56\x1f\x5c\x5c\x76\x98\x4e\xd5\xac\xa9\x5a\xfa\x02\xe1\x8b\x08\xa7\x60\x6c\x51\xd8\xfa\xdc\x63\x5d\x2a\xc0\xe3\x13\xee\x87\xc3\x20\x18\x04\x0f\xdf\x01\x00\x00\xff\xff\x5b\x77\xd0\x30\x30\x02\x00\x00")

func _20200319131542_create_layers_table_up_sql() ([]byte, error) {
	return bindata_read(
		__20200319131542_create_layers_table_up_sql,
		"20200319131542_create_layers_table.up.sql",
	)
}

var __20200319131632_create_manifest_layers_table_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\xc8\x4d\xcc\xcb\x4c\x4b\x2d\x2e\x89\xcf\x49\xac\x4c\x2d\x2a\xb6\x06\x04\x00\x00\xff\xff\xda\x9b\xec\x68\x25\x00\x00\x00")

func _20200319131632_create_manifest_layers_table_down_sql() ([]byte, error) {
	return bindata_read(
		__20200319131632_create_manifest_layers_table_down_sql,
		"20200319131632_create_manifest_layers_table.down.sql",
	)
}

var __20200319131632_create_manifest_layers_table_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x91\xc1\x4e\xb3\x40\x14\x85\xf7\x7d\x8a\xb3\x84\xa4\x6f\xf0\xaf\xa6\x70\x69\x26\x3f\x0e\x3a\x5c\x12\x59\x11\x94\xa9\x4e\x14\x5a\xcb\x98\x46\x9f\xde\x08\x02\xad\x18\xe5\xae\x20\x70\xce\xf9\xce\xbd\x81\x26\xc1\x04\x16\x9b\x98\x20\x23\xa8\x84\x41\xb7\x32\xe5\x14\x75\xd9\xd8\x9d\x69\x5d\xf1\x5c\xbe\x99\x63\xbb\xf2\x56\x00\x60\x2b\x8c\x73\x67\x1f\x6c\xe3\x30\x9f\x4f\x17\x95\xc5\x31\xb6\xa4\x48\x0b\xa6\x10\x9b\x1c\x21\x45\x22\x8b\x19\x22\x85\x0c\x49\xb1\xe4\x7c\xdd\x79\x8e\x49\xb6\xfa\xdb\xb3\x97\x74\x4c\x45\x0f\xb3\x50\x72\x7f\x34\xa5\x33\x55\x51\x3a\xc0\xd9\xda\xb4\xae\xac\x0f\x38\x59\xf7\xd8\xbd\xe2\x7d\xdf\x98\x89\x7c\x80\x6d\xf6\x27\xcf\xef\x0d\x82\x44\xa5\xac\x85\x54\x8c\xc3\x53\xf1\x6d\x3d\xb8\xd6\xf2\x4a\xe8\x1c\xff\x29\x87\x67\xab\xb9\x66\x37\xd3\x14\x67\xc5\xc7\xe7\x16\x51\xa2\x49\x6e\x55\xef\x74\xf6\x8b\xbf\x1a\x8a\x69\x8a\x48\x93\x0a\x68\xba\x52\xdb\x85\x22\x51\x08\x29\x26\x26\x04\x22\x0d\x44\x48\x4b\x30\x86\x65\x0e\x55\x2e\x00\x86\x8f\x3f\xa6\x7f\x09\x16\x46\xbf\xbe\xfc\xba\x81\xf1\xa6\x99\x92\x37\x19\x5d\x74\x5f\x63\xe2\xf0\xff\x7d\x04\x00\x00\xff\xff\xdb\x8f\x22\x29\xb6\x02\x00\x00")

func _20200319131632_create_manifest_layers_table_up_sql() ([]byte, error) {
	return bindata_read(
		__20200319131632_create_manifest_layers_table_up_sql,
		"20200319131632_create_manifest_layers_table.up.sql",
	)
}

var __20200319131907_create_manifest_lists_table_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\xc8\x4d\xcc\xcb\x4c\x4b\x2d\x2e\x89\xcf\xc9\x2c\x2e\x29\xb6\x06\x04\x00\x00\xff\xff\xce\xc8\x77\x1d\x24\x00\x00\x00")

func _20200319131907_create_manifest_lists_table_down_sql() ([]byte, error) {
	return bindata_read(
		__20200319131907_create_manifest_lists_table_down_sql,
		"20200319131907_create_manifest_lists_table.down.sql",
	)
}

var __20200319131907_create_manifest_lists_table_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x91\x41\x4f\x83\x30\x14\xc7\xef\xfb\x14\xff\x23\x24\x9e\x4c\x76\x52\x0f\x8c\x75\x4a\x86\x4c\xa1\x24\x72\x6a\x2a\x3c\xa1\xd9\x28\x08\xd5\x6d\x7e\x7a\xb3\x55\x65\x6e\x2e\xf1\xdd\x78\xbc\xff\xaf\xfd\xbd\xfa\x31\xf3\x38\x03\xf7\x26\x21\x43\x30\x43\xb4\xe0\x60\x4f\x41\xc2\x13\xd4\x52\xab\x17\xea\x8d\x58\xa9\xde\xf4\x23\x67\x04\x00\xaa\xc0\x61\x3d\xab\x52\x69\x83\xd3\xda\x71\xa2\x34\x0c\x71\xcb\x22\x16\x7b\x9c\x4d\x31\xc9\x30\x65\x33\x2f\x0d\x39\xbc\x04\xc1\x94\x45\x3c\xe0\xd9\xc5\x1e\x9b\x77\x24\x0d\x15\x42\x5a\x96\x51\x35\xf5\x46\xd6\x2d\xd6\xca\x54\xfb\x4f\x7c\x34\x9a\x06\xec\x37\x49\x37\x6b\xc7\xb5\x8c\x5a\x76\xcb\x01\x71\x96\x61\x87\xfb\xbc\xa2\x5a\x8a\x77\xea\x7a\xd5\x68\x28\x6d\xa8\xa4\xee\xbc\x87\x4d\x15\xaa\xdc\x2d\xa4\xa2\x8d\xb5\xdf\x1a\x92\x7f\xc8\x1f\xa5\x5a\xb9\x5d\x35\xf2\x67\x71\xff\x4c\xd5\x54\x28\x29\xcc\xb6\x25\xab\x43\x1b\x63\x7f\xf8\x8b\x28\xe1\xb1\x17\x44\x1c\xed\x52\xfc\x7e\x25\x3c\xc4\xc1\xbd\x17\x67\x98\xb3\x0c\x8e\x2a\xdc\x93\xc8\xdb\xeb\x51\x44\x1c\x48\xa5\x51\xf0\x98\x32\x38\x43\xeb\x14\x90\x1f\x9f\x29\x86\x9b\x8a\x15\xe9\xd2\x54\xf0\xef\x98\x3f\x87\xe3\xe4\x95\xec\xbe\x7a\xce\x30\xe5\xe2\xfa\x06\x97\xe3\xb1\xeb\x8e\xdc\xab\xcf\x00\x00\x00\xff\xff\x0b\x26\xa1\x12\x81\x02\x00\x00")

func _20200319131907_create_manifest_lists_table_up_sql() ([]byte, error) {
	return bindata_read(
		__20200319131907_create_manifest_lists_table_up_sql,
		"20200319131907_create_manifest_lists_table.up.sql",
	)
}

var __20200319132010_create_manifest_list_manifests_table_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\xc8\x4d\xcc\xcb\x4c\x4b\x2d\x2e\x89\xcf\xc9\x2c\x2e\x89\x87\xf1\x8a\xad\x01\x01\x00\x00\xff\xff\xc4\xb2\x07\x11\x2d\x00\x00\x00")

func _20200319132010_create_manifest_list_manifests_table_down_sql() ([]byte, error) {
	return bindata_read(
		__20200319132010_create_manifest_list_manifests_table_down_sql,
		"20200319132010_create_manifest_list_manifests_table.down.sql",
	)
}

var __20200319132010_create_manifest_list_manifests_table_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x91\x41\x4f\x83\x40\x10\x85\xef\xfd\x15\xef\x08\x49\xff\x81\xa7\x2d\x0c\xcd\x46\x5c\x74\x19\x12\x39\x11\x14\xaa\x1b\x85\x56\x77\x4d\x13\x7f\xbd\x91\x66\xa3\x40\x48\x9b\xce\x6d\x36\xf3\xde\x7e\xf3\x26\xd2\x24\x98\xc0\x62\x93\x12\x64\x02\x95\x31\xe8\x51\xe6\x9c\xa3\xab\x7b\xb3\x6b\xad\xab\xde\x8d\x75\x95\xef\xec\x2a\x58\x01\x80\x69\x30\xae\x27\xf3\x62\x7a\x87\x79\xfd\x5a\xaa\x22\x4d\xb1\x25\x45\x5a\x30\xc5\xd8\x94\x88\x29\x11\x45\xca\x10\x39\x64\x4c\x8a\x25\x97\xeb\xc1\x78\xfc\xad\x69\xce\x1b\x4f\x74\x9e\xec\x42\xdd\xf3\x67\x5b\xbb\xb6\xa9\x6a\x3f\xea\x4c\xd7\x5a\x57\x77\x07\x1c\x8d\x7b\x1d\x5a\x7c\xef\xfb\xf6\x6f\x11\xcf\xde\xef\x8f\x41\x78\x72\x89\x32\x95\xb3\x16\x52\x31\x0e\x6f\xd5\x42\x74\xb8\xd7\xf2\x4e\xe8\x12\xb7\x54\x22\x30\xcd\x5c\xbb\x5b\xd4\x56\xd3\x5c\xc6\x0f\x16\x49\xa6\x49\x6e\xd5\xc9\x7b\x3a\x1c\xae\xfc\xf2\x9a\x12\xd2\xa4\x22\x9a\x1c\xd8\x0e\x40\xc8\x14\x62\x4a\x89\x09\x91\xc8\x23\x11\xd3\x55\x88\xff\xe8\x96\xc0\xce\x30\x5d\x8c\xf3\xf5\x71\x45\x62\xa6\x41\xa1\xe4\x43\x41\xf3\xa4\xd6\x18\x21\x86\x37\x3f\x01\x00\x00\xff\xff\x72\xbc\x32\x18\x22\x03\x00\x00")

func _20200319132010_create_manifest_list_manifests_table_up_sql() ([]byte, error) {
	return bindata_read(
		__20200319132010_create_manifest_list_manifests_table_up_sql,
		"20200319132010_create_manifest_list_manifests_table.up.sql",
	)
}

var __20200319132237_create_tags_table_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\x28\x49\x4c\x2f\xb6\x06\x04\x00\x00\xff\xff\x83\x0d\x99\xe1\x1a\x00\x00\x00")

func _20200319132237_create_tags_table_down_sql() ([]byte, error) {
	return bindata_read(
		__20200319132237_create_tags_table_down_sql,
		"20200319132237_create_tags_table.down.sql",
	)
}

var __20200319132237_create_tags_table_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x93\x41\x8f\x9b\x30\x10\x85\xef\xfc\x8a\x77\xc4\xd2\x9e\x2a\xed\xa9\xed\xc1\x0b\x93\xad\xb5\xd4\xb4\xc6\x48\xcd\x09\xd1\xc4\x49\xac\xdd\x40\x1a\xbc\xda\xb6\xbf\xbe\x22\x26\x09\x84\x24\x8d\x96\x13\xb6\xdf\x7c\xf3\xc6\xe3\x89\x14\x71\x4d\xd0\xfc\x21\x21\x88\x09\x64\xaa\x41\x3f\x44\xa6\x33\xb8\x72\xd9\x04\x61\x00\x00\x76\x8e\xe1\xf7\xd3\x2e\x6d\xe5\x30\xfe\xda\x78\x99\x27\x09\x1e\x49\x92\xe2\x9a\x62\x3c\x4c\x11\xd3\x84\xe7\x89\x06\xcf\x20\x62\x92\x5a\xe8\xe9\xdd\x0e\xbc\x35\x9b\xba\xb1\xae\xde\xfe\x29\x7c\x8e\xff\x82\x7d\xdc\xba\xac\xec\xc2\x34\xae\xd8\x3b\xf3\x71\x27\x87\x2f\xd6\x2b\xfa\x87\xb3\xad\x29\x9d\x99\x17\xe5\x3e\x89\xb3\x6b\xd3\xb8\x72\xbd\xc1\x9b\x75\xab\xdd\x12\x7f\xeb\xca\x1c\x4b\xd9\xbb\xaf\xea\xb7\x90\x79\xca\xeb\x66\x7e\x1b\xc5\xcb\xab\x72\x6d\x06\xd5\x38\xf3\xfb\x5c\x91\xa7\x65\x46\xa9\xcc\xb4\xe2\x42\x6a\x6c\x9e\x8b\xb6\x21\xf8\xa6\xc4\x57\xae\xa6\x78\xa2\x29\x42\x3b\x67\x23\xe1\xc2\x0b\x8b\xc1\xd5\x1e\x57\xd6\x34\x98\xa4\x8a\xc4\xa3\xf4\x8c\x81\x8e\x05\x7b\x23\x8a\x26\xa4\x48\x46\x94\x61\x10\xdb\xe6\x44\x2a\x11\x53\x42\x9a\x10\xf1\x2c\xe2\x31\x5d\x74\xd1\x6b\xd4\xe1\xff\xc4\x40\x4f\x72\x36\xfd\x31\xec\xbd\xb9\xbb\x77\x30\xdc\xb8\xe4\xa2\x13\x5f\xb5\xd2\x01\x6e\xf4\xf3\xfa\xcb\xfb\x69\x5f\xc1\xb0\x2d\xc8\xa5\xf8\x9e\x13\xc2\xf6\xe8\x6e\x38\x0d\xe3\xce\xce\x9e\x7b\x9c\x17\x53\x2d\xdd\x0a\xd1\x17\x8a\x9e\x10\x86\xb3\x55\xb9\xed\xf6\x76\x30\x86\x4f\x9f\xf1\xe1\xfe\x9e\x5d\xc6\x9c\x6b\xcd\xe1\xaa\xea\xca\x14\xf5\xe2\x80\x0f\xfa\x6f\xb4\xdf\x31\x88\xec\x38\x28\x5c\xc6\xe3\xe1\x6b\x05\x79\x92\x30\xa4\xea\x3a\xe5\x2a\xa1\x4b\xc1\x18\x0b\xd8\xc7\x7f\x01\x00\x00\xff\xff\x6e\x5f\x51\xbd\xb5\x04\x00\x00")

func _20200319132237_create_tags_table_up_sql() ([]byte, error) {
	return bindata_read(
		__20200319132237_create_tags_table_up_sql,
		"20200319132237_create_tags_table.up.sql",
	)
}

var __20200408192311_create_repository_manifests_table_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\x28\x4a\x2d\xc8\x2f\xce\x2c\xc9\x2f\xaa\x8c\xcf\x4d\xcc\xcb\x4c\x4b\x2d\x2e\x29\xb6\x06\x04\x00\x00\xff\xff\xf7\xde\xbc\xab\x2a\x00\x00\x00")

func _20200408192311_create_repository_manifests_table_down_sql() ([]byte, error) {
	return bindata_read(
		__20200408192311_create_repository_manifests_table_down_sql,
		"20200408192311_create_repository_manifests_table.down.sql",
	)
}

var __20200408192311_create_repository_manifests_table_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x92\xdf\x6a\xb4\x30\x14\xc4\xef\xf7\x29\xe6\x72\x85\x7d\x89\xac\x1e\x97\xf0\xf9\xc5\x36\x46\xa8\x57\x62\x6b\xb6\x0d\xc5\x3f\xd5\x94\xa5\x7d\xfa\xa2\x22\xc6\xae\x50\xcf\xdd\x81\x99\xf1\x77\xc6\xf8\x92\x98\x22\x28\x76\x8e\x08\x3c\x84\x88\x15\xe8\x89\x27\x2a\x41\xa7\xdb\xa6\x37\xb6\xe9\xbe\xf2\xaa\xa8\xcd\x55\xf7\xb6\x3f\x1c\x0f\x00\x60\x4a\x38\xf3\x6c\x5e\x4d\x6d\x71\x3f\x43\x98\x48\xa3\x08\x17\x12\x24\x99\xa2\x00\xe7\x0c\x01\x85\x2c\x8d\x14\x58\x02\x1e\x90\x50\x5c\x65\xa7\x31\xd5\xf9\xa0\x29\xff\x4e\x9d\x4c\x33\x5a\x3e\x32\xed\x34\xbd\x74\xba\xb0\xba\xcc\x8b\x51\x6a\x4d\xa5\x7b\x5b\x54\x2d\x6e\xc6\xbe\x8d\x2b\xbe\x9b\x5a\x2f\xfc\x33\x72\xdd\xdc\x8e\xde\x14\xe1\xc7\x22\x51\x92\x71\xa1\xd0\xbe\xe7\x5b\x5d\xe1\x41\xf2\xff\x4c\x66\xf8\x47\x19\x8e\xa6\xbc\x37\x5e\xb7\x8d\xf9\xaa\x88\x65\x33\xba\x47\x18\x4b\xe2\x17\x31\x65\xae\x74\xde\x61\xbe\x55\x52\x48\x92\x84\x4f\xce\x3f\x1c\xbc\x03\x03\x62\x81\x80\x22\x52\x04\x9f\x25\x3e\x0b\x68\x37\x95\xd3\xb4\x73\xe4\x0a\xc8\x91\x6c\xe2\x2c\xb6\x9d\x2c\x9f\x1f\x7b\x1a\x72\xdf\x40\x2a\xf8\x63\x4a\xbf\xba\x39\x61\x45\xe6\xfd\x04\x00\x00\xff\xff\x4d\x19\x6d\xa5\xf6\x02\x00\x00")

func _20200408192311_create_repository_manifests_table_up_sql() ([]byte, error) {
	return bindata_read(
		__20200408192311_create_repository_manifests_table_up_sql,
		"20200408192311_create_repository_manifests_table.up.sql",
	)
}

var __20200408193126_create_repository_manifest_lists_table_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\x28\x4a\x2d\xc8\x2f\xce\x2c\xc9\x2f\xaa\x8c\xcf\x4d\xcc\xcb\x4c\x4b\x2d\x2e\x89\xcf\xc9\x2c\x2e\x29\xb6\x06\x04\x00\x00\xff\xff\xd8\xa5\xd5\x69\x2f\x00\x00\x00")

func _20200408193126_create_repository_manifest_lists_table_down_sql() ([]byte, error) {
	return bindata_read(
		__20200408193126_create_repository_manifest_lists_table_down_sql,
		"20200408193126_create_repository_manifest_lists_table.down.sql",
	)
}

var __20200408193126_create_repository_manifest_lists_table_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x94\x92\xc1\x4e\xf3\x30\x10\x84\xef\x79\x8a\x39\x36\x52\xdf\xe0\x3f\xb9\xc9\xa6\xb2\xfe\xe0\x80\xe3\x08\x72\x8a\x02\x75\xc1\x82\x36\x21\x36\x42\xf0\xf4\x28\x8d\xa2\x36\x4e\x81\xe2\xdb\x4a\x3b\xb3\xdf\x8e\x37\x92\xc4\x14\x41\xb1\x55\x4a\xe0\x09\x44\xa6\x40\x77\x3c\x57\x39\x3a\xdd\x36\xd6\xb8\xa6\xfb\xa8\x76\xf5\xde\x6c\xb5\x75\xd5\x8b\xb1\xce\x06\x8b\x00\x00\xcc\x06\xd3\x77\x6f\x1e\xcd\xde\x61\xfe\x7a\x53\x51\xa4\x29\xd6\x24\x48\x32\x45\x31\x56\x25\x62\x4a\x58\x91\x2a\xb0\x1c\x3c\x26\xa1\xb8\x2a\x97\x07\xe3\x93\xc1\xc3\x8c\x5f\x8d\x07\xdd\x84\xb2\x97\x5e\xa8\x7b\xe8\x74\xed\xf4\xa6\xaa\xc7\x56\x67\x76\xda\xba\x7a\xd7\xe2\xdd\xb8\xa7\x43\x89\xcf\x66\xaf\x8f\x8b\x8c\xec\x22\xbb\x5d\x84\x83\x4b\x94\x89\x5c\x49\xc6\x85\x42\xfb\x5c\x7d\x1b\x1e\xae\x25\xbf\x62\xb2\xc4\x7f\x2a\xb1\x30\x9b\xb9\x7a\xfb\x83\xba\x9a\x64\x73\xac\x8c\xb6\x48\x32\x49\x7c\x2d\x06\xe3\x49\x5f\x18\x8c\x8b\x4b\x4a\x48\x92\x88\xe8\xe4\x7b\x7b\x6d\x0f\x82\x4c\x20\xa6\x94\x14\x21\x62\x79\xc4\x62\xfa\x1b\x9a\x1f\xbf\xbf\xf8\x84\xcf\x6f\x3e\x8b\xe8\x19\x5c\x08\xf9\xf6\x7a\x0e\xd2\x8f\x6e\x76\x2c\x85\xe0\x37\x05\x79\xc9\x2d\x67\x47\x15\x06\xe1\xbf\xaf\x00\x00\x00\xff\xff\x8b\xb6\x66\x2d\x35\x03\x00\x00")

func _20200408193126_create_repository_manifest_lists_table_up_sql() ([]byte, error) {
	return bindata_read(
		__20200408193126_create_repository_manifest_lists_table_up_sql,
		"20200408193126_create_repository_manifest_lists_table.up.sql",
	)
}

var __20200428184744_create_foreign_key_indexes_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x72\x75\xf7\xf4\xb3\xe6\x72\x09\xf2\x0f\x50\xf0\xf4\x73\x71\x8d\x50\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\xc8\xac\x88\x2f\x4a\x2d\xc8\x2f\xce\x2c\xc9\x2f\xca\x4c\x2d\x8e\x2f\x48\x2c\x4a\xcd\x2b\x89\xcf\x4c\xc1\xad\x3e\x37\x31\x2f\x33\x2d\xb5\xb8\x24\x3e\x27\xb1\x32\xb5\xa8\x18\xc1\x27\x45\x13\x98\x22\x52\x47\x66\x71\x09\x9c\x57\x8c\x26\x4e\x91\x09\xf8\x34\x97\x24\xa6\x17\x23\x82\xa6\x92\xb0\x5a\xe2\x4d\x25\xde\x07\x48\xf6\x23\x9c\x4f\xa4\xa3\xb0\xea\x25\xca\x91\x58\x74\x82\x9d\x4a\x89\xd5\x50\x03\x30\xbd\xee\xec\xef\xeb\xeb\x19\x62\x0d\x08\x00\x00\xff\xff\x1f\x19\xa0\xdd\xa4\x02\x00\x00")

func _20200428184744_create_foreign_key_indexes_down_sql() ([]byte, error) {
	return bindata_read(
		__20200428184744_create_foreign_key_indexes_down_sql,
		"20200428184744_create_foreign_key_indexes.down.sql",
	)
}

var __20200428184744_create_foreign_key_indexes_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xac\x91\xbd\x0e\x82\x30\x14\x85\x77\x9f\xe2\x8e\xf8\x0c\x4c\xfe\x54\xd3\x81\x92\x48\x07\xb6\xa6\x89\xd5\x34\x51\x20\x6d\x07\x79\x7b\x23\x20\x94\x5a\xb5\x8d\x4e\x84\x7b\x6e\xbe\xd3\x73\xee\x1a\xed\x31\x49\x17\x9b\x03\x5a\x51\x04\x98\x6c\x51\x09\x78\x07\x24\xa7\x80\x4a\x5c\xd0\x02\xe4\x8d\x29\xd1\xd4\x5a\x9a\x5a\x49\xa1\x59\xc3\x95\xa8\x0c\x93\x47\xc8\x09\xd8\x0a\x24\xa3\xb4\xfc\x46\xbc\xf2\x4a\x9e\x84\x36\xec\xc2\x5b\xa1\xf4\xf4\xdf\x63\x1d\x19\x12\x4b\x8f\x66\x77\x9f\x77\xe0\xa7\x18\x41\x95\xda\x8c\x7f\xda\x99\xbb\x2e\xb3\x5d\x2b\xc6\xb0\xfc\xbb\x6b\xa8\x61\x80\x97\xe1\x67\x3d\x9d\xba\x1d\xd0\x8f\x29\x24\xb3\x71\x18\xc9\x79\x62\xcf\x89\x7e\x8f\xaf\x5e\x07\x15\xda\xa5\x95\x61\x2a\xf2\x25\xaf\x6f\x2b\x36\xbf\xd7\xc9\xe9\xc3\xef\x13\xd3\x8f\x87\xd0\x75\x11\x16\xaa\x5f\xfd\x43\xb2\xc1\xd3\x77\xa8\x0f\xb6\xbe\xeb\xe5\x59\x86\x69\x7a\x0f\x00\x00\xff\xff\x6c\x72\x87\xd3\x92\x04\x00\x00")

func _20200428184744_create_foreign_key_indexes_up_sql() ([]byte, error) {
	return bindata_read(
		__20200428184744_create_foreign_key_indexes_up_sql,
		"20200428184744_create_foreign_key_indexes.up.sql",
	)
}

var __20200527132906_create_repository_layers_table_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\x28\x4a\x2d\xc8\x2f\xce\x2c\xc9\x2f\xaa\x8c\xcf\x49\xac\x4c\x2d\x2a\xb6\x06\x04\x00\x00\xff\xff\xaa\xec\x88\x39\x27\x00\x00\x00")

func _20200527132906_create_repository_layers_table_down_sql() ([]byte, error) {
	return bindata_read(
		__20200527132906_create_repository_layers_table_down_sql,
		"20200527132906_create_repository_layers_table.down.sql",
	)
}

var __20200527132906_create_repository_layers_table_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x91\x51\x4f\x83\x30\x14\x85\xdf\xf7\x2b\xce\xe3\x96\xec\x1f\xf8\xd4\xc1\x65\x69\xc4\xa2\xa5\x24\xf2\x44\x50\xaa\x36\xba\x81\x50\xb3\xcc\x5f\x6f\x00\x2b\x20\x4b\x46\xdf\x6e\x7a\xcf\xb9\xdf\x3d\xd7\x93\xc4\x14\x41\xb1\x5d\x48\xe0\x01\x44\xa4\x40\x8f\x3c\x56\x31\x6a\x5d\x95\x8d\xb1\x65\x7d\xce\x3e\xf2\xb3\xae\x9b\xd5\x7a\x05\x00\xa6\xc0\xe8\x3d\x99\x57\x73\xb4\x98\xbf\xd6\x49\x24\x61\x88\x3d\x09\x92\x4c\x91\x8f\x5d\x0a\x9f\x02\x96\x84\x0a\x2c\x06\xf7\x49\x28\xae\xd2\x6d\xe7\x3a\x9a\x66\x8a\xeb\xae\xbd\xa8\xe3\xca\x1c\xd0\x42\xd1\x73\xad\x73\xab\x8b\x2c\xef\x5a\xad\x39\xe8\xc6\xe6\x87\x0a\x27\x63\xdf\xba\x12\xdf\xe5\x51\x0f\xfc\x0e\xf9\x58\x9e\xd6\x9b\xde\xc2\x8b\x44\xac\x24\xe3\x42\xa1\x7a\xcf\x66\x41\xe1\x5e\xf2\x3b\x26\x53\xdc\x52\x8a\xb5\x29\xe6\xaa\x97\x0b\xaa\x6c\x12\xc1\x50\x19\xdd\x20\x88\x24\xf1\xbd\xe8\x0d\x27\x7d\x9b\x95\xdb\x52\x52\x40\x92\x84\x47\xa3\xd3\xb5\xda\x16\x00\x91\x80\x4f\x21\x29\x82\xc7\x62\x8f\xf9\xb4\x0c\xc9\x05\xec\x16\x9b\x70\xb8\xcf\x8b\x08\xbf\x82\x85\xc3\xbf\x3e\xaf\xe6\xf1\x77\xeb\x44\xf0\x87\x84\xfe\xc5\xb0\xc5\x40\xb3\xb9\xf9\x09\x00\x00\xff\xff\xe3\x6a\x5c\xbd\xd6\x02\x00\x00")

func _20200527132906_create_repository_layers_table_up_sql() ([]byte, error) {
	return bindata_read(
		__20200527132906_create_repository_layers_table_up_sql,
		"20200527132906_create_repository_layers_table.up.sql",
	)
}

var __20200527133232_create_repository_layers_indexes_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x72\x75\xf7\xf4\xb3\xe6\x72\x09\xf2\x0f\x50\xf0\xf4\x73\x71\x8d\x50\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\xc8\xac\x88\x2f\x4a\x2d\xc8\x2f\xce\x2c\xc9\x2f\xaa\x8c\xcf\x49\xac\x4c\x2d\x2a\x46\x16\xc9\x4c\x21\x45\x23\x98\x02\xeb\x71\xf6\xf7\xf5\xf5\x0c\xb1\x06\x04\x00\x00\xff\xff\x71\xb0\x63\x1c\x7b\x00\x00\x00")

func _20200527133232_create_repository_layers_indexes_down_sql() ([]byte, error) {
	return bindata_read(
		__20200527133232_create_repository_layers_indexes_down_sql,
		"20200527133232_create_repository_layers_indexes.down.sql",
	)
}

var __20200527133232_create_repository_layers_indexes_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x72\x75\xf7\xf4\xb3\xe6\x72\x0e\x72\x75\x0c\x71\x55\xf0\xf4\x73\x71\x8d\x50\xf0\x74\x53\xf0\xf3\x0f\x51\x70\x8d\xf0\x0c\x0e\x09\x56\xc8\xac\x88\x2f\x4a\x2d\xc8\x2f\xce\x2c\xc9\x2f\xaa\x8c\xcf\x49\xac\x4c\x2d\x2a\x46\x16\xc9\x4c\x51\xf0\xf7\x53\xc0\x50\xa2\xa0\x81\xa2\x46\x93\x74\x3b\xc0\x14\x6e\xe3\x61\xd2\x20\x93\xfd\x7d\x7d\x3d\x43\xac\x01\x01\x00\x00\xff\xff\xf3\x6c\x4b\xda\xcc\x00\x00\x00")

func _20200527133232_create_repository_layers_indexes_up_sql() ([]byte, error) {
	return bindata_read(
		__20200527133232_create_repository_layers_indexes_up_sql,
		"20200527133232_create_repository_layers_indexes.up.sql",
	)
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		return f()
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() ([]byte, error){
	"20200319122755_create_repositories_table.down.sql":              _20200319122755_create_repositories_table_down_sql,
	"20200319122755_create_repositories_table.up.sql":                _20200319122755_create_repositories_table_up_sql,
	"20200319130108_create_manifests_table.down.sql":                 _20200319130108_create_manifests_table_down_sql,
	"20200319130108_create_manifests_table.up.sql":                   _20200319130108_create_manifests_table_up_sql,
	"20200319131222_create_manifest_configurations_table.down.sql":   _20200319131222_create_manifest_configurations_table_down_sql,
	"20200319131222_create_manifest_configurations_table.up.sql":     _20200319131222_create_manifest_configurations_table_up_sql,
	"20200319131542_create_layers_table.down.sql":                    _20200319131542_create_layers_table_down_sql,
	"20200319131542_create_layers_table.up.sql":                      _20200319131542_create_layers_table_up_sql,
	"20200319131632_create_manifest_layers_table.down.sql":           _20200319131632_create_manifest_layers_table_down_sql,
	"20200319131632_create_manifest_layers_table.up.sql":             _20200319131632_create_manifest_layers_table_up_sql,
	"20200319131907_create_manifest_lists_table.down.sql":            _20200319131907_create_manifest_lists_table_down_sql,
	"20200319131907_create_manifest_lists_table.up.sql":              _20200319131907_create_manifest_lists_table_up_sql,
	"20200319132010_create_manifest_list_manifests_table.down.sql":   _20200319132010_create_manifest_list_manifests_table_down_sql,
	"20200319132010_create_manifest_list_manifests_table.up.sql":     _20200319132010_create_manifest_list_manifests_table_up_sql,
	"20200319132237_create_tags_table.down.sql":                      _20200319132237_create_tags_table_down_sql,
	"20200319132237_create_tags_table.up.sql":                        _20200319132237_create_tags_table_up_sql,
	"20200408192311_create_repository_manifests_table.down.sql":      _20200408192311_create_repository_manifests_table_down_sql,
	"20200408192311_create_repository_manifests_table.up.sql":        _20200408192311_create_repository_manifests_table_up_sql,
	"20200408193126_create_repository_manifest_lists_table.down.sql": _20200408193126_create_repository_manifest_lists_table_down_sql,
	"20200408193126_create_repository_manifest_lists_table.up.sql":   _20200408193126_create_repository_manifest_lists_table_up_sql,
	"20200428184744_create_foreign_key_indexes.down.sql":             _20200428184744_create_foreign_key_indexes_down_sql,
	"20200428184744_create_foreign_key_indexes.up.sql":               _20200428184744_create_foreign_key_indexes_up_sql,
	"20200527132906_create_repository_layers_table.down.sql":         _20200527132906_create_repository_layers_table_down_sql,
	"20200527132906_create_repository_layers_table.up.sql":           _20200527132906_create_repository_layers_table_up_sql,
	"20200527133232_create_repository_layers_indexes.down.sql":       _20200527133232_create_repository_layers_indexes_down_sql,
	"20200527133232_create_repository_layers_indexes.up.sql":         _20200527133232_create_repository_layers_indexes_up_sql,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func     func() ([]byte, error)
	Children map[string]*_bintree_t
}

var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"20200319122755_create_repositories_table.down.sql":              {_20200319122755_create_repositories_table_down_sql, map[string]*_bintree_t{}},
	"20200319122755_create_repositories_table.up.sql":                {_20200319122755_create_repositories_table_up_sql, map[string]*_bintree_t{}},
	"20200319130108_create_manifests_table.down.sql":                 {_20200319130108_create_manifests_table_down_sql, map[string]*_bintree_t{}},
	"20200319130108_create_manifests_table.up.sql":                   {_20200319130108_create_manifests_table_up_sql, map[string]*_bintree_t{}},
	"20200319131222_create_manifest_configurations_table.down.sql":   {_20200319131222_create_manifest_configurations_table_down_sql, map[string]*_bintree_t{}},
	"20200319131222_create_manifest_configurations_table.up.sql":     {_20200319131222_create_manifest_configurations_table_up_sql, map[string]*_bintree_t{}},
	"20200319131542_create_layers_table.down.sql":                    {_20200319131542_create_layers_table_down_sql, map[string]*_bintree_t{}},
	"20200319131542_create_layers_table.up.sql":                      {_20200319131542_create_layers_table_up_sql, map[string]*_bintree_t{}},
	"20200319131632_create_manifest_layers_table.down.sql":           {_20200319131632_create_manifest_layers_table_down_sql, map[string]*_bintree_t{}},
	"20200319131632_create_manifest_layers_table.up.sql":             {_20200319131632_create_manifest_layers_table_up_sql, map[string]*_bintree_t{}},
	"20200319131907_create_manifest_lists_table.down.sql":            {_20200319131907_create_manifest_lists_table_down_sql, map[string]*_bintree_t{}},
	"20200319131907_create_manifest_lists_table.up.sql":              {_20200319131907_create_manifest_lists_table_up_sql, map[string]*_bintree_t{}},
	"20200319132010_create_manifest_list_manifests_table.down.sql":   {_20200319132010_create_manifest_list_manifests_table_down_sql, map[string]*_bintree_t{}},
	"20200319132010_create_manifest_list_manifests_table.up.sql":     {_20200319132010_create_manifest_list_manifests_table_up_sql, map[string]*_bintree_t{}},
	"20200319132237_create_tags_table.down.sql":                      {_20200319132237_create_tags_table_down_sql, map[string]*_bintree_t{}},
	"20200319132237_create_tags_table.up.sql":                        {_20200319132237_create_tags_table_up_sql, map[string]*_bintree_t{}},
	"20200408192311_create_repository_manifests_table.down.sql":      {_20200408192311_create_repository_manifests_table_down_sql, map[string]*_bintree_t{}},
	"20200408192311_create_repository_manifests_table.up.sql":        {_20200408192311_create_repository_manifests_table_up_sql, map[string]*_bintree_t{}},
	"20200408193126_create_repository_manifest_lists_table.down.sql": {_20200408193126_create_repository_manifest_lists_table_down_sql, map[string]*_bintree_t{}},
	"20200408193126_create_repository_manifest_lists_table.up.sql":   {_20200408193126_create_repository_manifest_lists_table_up_sql, map[string]*_bintree_t{}},
	"20200428184744_create_foreign_key_indexes.down.sql":             {_20200428184744_create_foreign_key_indexes_down_sql, map[string]*_bintree_t{}},
	"20200428184744_create_foreign_key_indexes.up.sql":               {_20200428184744_create_foreign_key_indexes_up_sql, map[string]*_bintree_t{}},
	"20200527132906_create_repository_layers_table.down.sql":         {_20200527132906_create_repository_layers_table_down_sql, map[string]*_bintree_t{}},
	"20200527132906_create_repository_layers_table.up.sql":           {_20200527132906_create_repository_layers_table_up_sql, map[string]*_bintree_t{}},
	"20200527133232_create_repository_layers_indexes.down.sql":       {_20200527133232_create_repository_layers_indexes_down_sql, map[string]*_bintree_t{}},
	"20200527133232_create_repository_layers_indexes.up.sql":         {_20200527133232_create_repository_layers_indexes_up_sql, map[string]*_bintree_t{}},
}}
