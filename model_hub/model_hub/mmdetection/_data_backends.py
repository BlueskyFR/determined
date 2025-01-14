"""
Add backends to support loading data from other sources including
S3 buckets, GCS storage buckets, and fake data.
"""
import logging
import os
import time
from typing import Any, Dict, List, Optional, Union, cast

import boto3
import google.cloud as gcs
import mmcv

import determined
import model_hub.utils as utils


class S3Backend(mmcv.fileio.BaseStorageBackend):  # type: ignore
    def __init__(self, bucket_name: str):
        self._storage_client = boto3.client("s3")
        self._bucket = bucket_name

    def get(self, filepath: str) -> Any:
        obj = self._storage_client.get_object(Bucket=self._bucket, Key=filepath)
        img_str = obj["Body"].read()
        return img_str

    def get_text(self, filepath: str) -> Any:
        raise NotImplementedError


mmcv.fileio.FileClient.register_backend("s3", S3Backend)


class GCSBackend(mmcv.fileio.BaseStorageBackend):  # type: ignore
    def __init__(self, bucket_name: str):
        self._storage_client = gcs.storage.Client()
        self._bucket = self._storage_client.bucket(bucket_name)

    def get(self, filepath: str) -> Any:
        blob = self._bucket.blob(filepath)
        try:
            img_str = determined.util.download_gcs_blob_with_backoff(blob)
        except Exception as e:
            raise Exception(f"Encountered {e}, failed to download {filepath} from gcs bucket.")
        return img_str

    def get_text(self, filepath: str) -> Any:
        raise NotImplementedError


mmcv.fileio.FileClient.register_backend("gcs", GCSBackend)


class FakeBackend(mmcv.fileio.BaseStorageBackend):  # type: ignore
    def __init__(self, fake_img_path: Optional[str] = None):
        if fake_img_path is None:
            download_dir = os.path.join("/tmp", str(hash(time.time())))
            os.makedirs(download_dir, exist_ok=True)
            fake_img_path = utils.download_url(
                download_dir,
                "https://images.freeimages.com/images/large-previews/5c6/sunset-jungle-1383333.jpg",
            )
            logging.info("Downloaded fake image to use.")

        with open(fake_img_path, "rb") as f:
            img_str = f.read()
        self.data = img_str

    def get(self, filepath: str) -> Any:
        return self.data

    def get_text(self, filepath: str) -> Any:
        raise NotImplementedError


mmcv.fileio.FileClient.register_backend("fake", FakeBackend)


def sub_backend(
    file_client_args: Dict[str, Any],
    cfg: Union[mmcv.utils.config.Config, mmcv.utils.config.ConfigDict, List],
) -> None:
    """
    Recurses through the mmcv.Config to replace the `file_client_args` field of calls to
    `LoadImageFromFile` with the provided argument.  `file_client_args` should be a dictionary
    with a `backend` specified along with keyword arguments to instantiate the backend.

    .. code-block:: python
        # Using gcs backend
        file_client_args = {'backend': 'gcs', 'bucket_name': 'mydatabucket'}
        # Using s3 backend
        file_client_args = {'backend': 's3', 'bucket_name': 'mydatabucket'}
        # Using fake backend
        file_client_args = {'backend': 'fake', 'fake_img_path': None}

    In addition to the backends registered in this file, mmcv supports
    disk, ceph, memcache, lmdb, petrel, and http backends. The default backend is disk.

    It is better to override the backend using this function than to use other mechanisms
    in `MMDetTrial.build_mmdet_config` because recursively going through the config will
    cover all occurrences of `LoadImageFromFile`.

    Arguments:
        file_client_args: dictionary with a backend field and keyword arguments for that backend.
        cfg: base config for which to replace backends.
    """
    if type(cfg) in [mmcv.utils.config.Config, mmcv.utils.config.ConfigDict]:
        cfg = cast(Union[mmcv.utils.config.Config, mmcv.utils.config.ConfigDict], cfg)
        if "type" in cfg and cfg["type"] == "LoadImageFromFile":
            cfg["file_client_args"] = file_client_args
        else:
            for k in cfg:
                sub_backend(file_client_args, cfg[k])
    else:
        if isinstance(cfg, list):
            for i in cfg:
                sub_backend(file_client_args, i)
