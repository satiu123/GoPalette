import { blob } from 'hub:blob'
import type { BlobSize } from '@nuxthub/core/blob'

const NUXTHUB_IMAGE_UPLOAD_MAX_SIZE = '8MB' satisfies BlobSize
const STRICT_IMAGE_MAX_BYTES = 5 * 1024 * 1024

export default eventHandler(async (event) => {
  const contentLength = Number(getRequestHeader(event, 'content-length') || 0)
  if (Number.isFinite(contentLength) && contentLength > STRICT_IMAGE_MAX_BYTES) {
    throw createError({
      statusCode: 413,
      statusMessage: '图片过大',
      message: '图片大小不能超过 5MB'
    })
  }

  return blob.handleUpload(event, {
    formKey: 'file',
    multiple: false,
    ensure: {
      maxSize: NUXTHUB_IMAGE_UPLOAD_MAX_SIZE,
      types: ['image']
    },
    put: {
      addRandomSuffix: true
    }
  })
})
