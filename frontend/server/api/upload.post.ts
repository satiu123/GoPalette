import { blob } from 'hub:blob'
import type { BlobSize } from '@nuxthub/core/blob'

const IMAGE_UPLOAD_MAX_SIZE = '5MB' as BlobSize

export default eventHandler(async (event) => {
  return blob.handleUpload(event, {
    formKey: 'file',
    multiple: false,
    ensure: {
      maxSize: IMAGE_UPLOAD_MAX_SIZE,
      types: ['image']
    },
    put: {
      addRandomSuffix: true
    }
  })
})
