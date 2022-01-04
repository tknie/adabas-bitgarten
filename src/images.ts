import axios from 'axios';
import { authHeader } from './auth-header';
import store from './store';
import { config } from './config';
import { userService } from './user.service';

export const image = {
    loadImage,
    loadVideo,
    loadThumbnail,
    loadPictureBases,
    loadPictureDirectory,
};

export async function loadImage(md5: string) {
    const x = store.getters.getImageByMd5(md5);
    if (x) {
        return Promise.resolve({ data: x });
    }
    // console.log('Loading image MD5=' + md5);
    return await axios({
        // example url
        url: config.Url() +
            '/binary/map/Picture/*/Media?search=Md5=' +
            md5,
        method: 'GET',
        headers: authHeader(''),
        responseType: 'arraybuffer',
    }).then((response: any) => {
        const bytes = new Uint8Array(response.data);
        const binary = bytes.reduce((data, b) => data += String.fromCharCode(b), '');
        response.data = 'data:image/jpeg;base64,' + btoa(binary);
        const img = new Image();
        const i = {
            md5, width: 0, height: 0, fill: 'fill',
            MIMEType: 'image/jpeg', src: response.data, time: new Date(),
        };
        img.onload = () => {
            i.width = img.width;
            i.height = img.height;
            // console.log('Onload '+img.width+' '+img.height+' '+i.md5);
            store.commit('ADD_IMAGE', i);
        };
        img.src = response.data;
        return response.data;
    },
        (error: any) => console.log('Image read error ' + md5 + ': ' + error));
}

export async function loadPictureBases() {
    const requestOptions = {
        method: 'GET',
        headers: authHeader('application/json'),
    };
    // console.log('Loading picture base');
    const response = await fetch(`${config.Url()}/rest/map/PictureMetadata?limit=0&start=0&descriptor=true&fields=Directory`,
        requestOptions).then(handleResponse);
    return response;
}

export async function loadPictureDirectory(directory: string) {
    const requestOptions = {
        method: 'GET',
        headers: authHeader('application/json'),
    };
    // console.log('Loading picture directory: ' + directory);
    const ret: any = await fetch(`${config.Url()}/rest/map/PictureMetadata?limit=0&search=Directory=` + directory,
        requestOptions).then((response: any) => {
            return response.text().then((element: any) => {
                // console.log('PD:' + element);
                const data = element && JSON.parse(element);
                if (!response.ok) {
                    if (response.status === 401 || response.status === 404) {
                        // auto logout if 401 response returned from api
                        userService.logout();
                        location.reload();
                    }

                    const error = (data && data.message) || response.statusText;
                    return Promise.reject(error);
                }
                // console.log('DATA:' + data);
                const p: any[] = [];
                data.Records.forEach((d: any) => {
                    p.push({ title: d.PictureName, msrc: d.Md5, index: d.ISN });
                });
                // console.log('Result ' + JSON.stringify(p));
                return p;
            });
        },
            (error: any) => console.log('Picture loading directory: ' + error));
    return ret;
}

function handleResponse(response: any) {
    return response.text().then((text: any) => {
        const data = text && JSON.parse(text);
        if (!response.ok) {
            if (response.status === 401 || response.status === 404) {
                // auto logout if 401 response returned from api
                userService.logout();
                location.reload();
            }

            const error = (data && data.message) || response.statusText;
            return Promise.reject(error);
        }
        const p: any[] = [];
        data.Records.forEach((element: any) => {
            p.push(element.Directory);
        });
        // console.log('Result ' + JSON.stringify(p));
        return p;
    });
}

export async function loadVideo(md5: string) {
    const x = store.getters.getImageByMd5(md5);
    if (x) {
        return Promise.resolve({ data: x });
    }
    // console.log('Loading image MD5=' + md5);
    return await axios({
        // example url
        url: config.Url() +
            '/video/map/PictureBinary/*/Media?mimetypeField=MIMEType&search=Md5=' +
            md5,
        method: 'GET',
        headers: authHeader(''),
        responseType: 'arraybuffer',
    }).then((response: any) => {
        const bytes = new Uint8Array(response.data);
        const binary = bytes.reduce((data, b) => data += String.fromCharCode(b), '');
        response.data = 'data:video/mp4;base64,' + btoa(binary);
        const i = {
            md5, width: 0, height: 0, MIMEType: 'video/mp4',
            src: response.data, fill: 'fillHeight', time: new Date(),
        };
        store.commit('ADD_IMAGE', i);
        return response.data;
    },
        (error: any) => console.log('Image read error ' + md5 + ': ' + error));
}

export async function loadThumbnail(md5: string) {
    const x = store.getters.getThumbnailByMd5(md5);
    if (x) {
        return Promise.resolve({ data: x });
    }
    // console.log('Not in cache, loading thumbnail ' + md5);
    return await axios({
        // example url
        url: config.Url() +
            '/binary/map/Picture/*/Thumbnail?search=Md5=' +
            md5,
        method: 'GET',
        headers: authHeader(''),
        responseType: 'arraybuffer',
    }).then((response: any) => {
        const bytes = new Uint8Array(response.data);
        const binary = bytes.reduce((data, b) => data += String.fromCharCode(b), '');
        response.data = 'data:image/jpeg;base64,' + btoa(binary);
        const i = { md5, src: response.data };
        store.commit('ADD_THUMB', i);
        return response.data;
    },
        (error: any) => console.log('Image read error ' + md5 + ': ' + error));
}

export async function streamVideo(md5: string, videoElement: any) {
    console.log('Streaming Video MD5=' + md5 + ' -> ' + JSON.stringify(videoElement) + ' ' + videoElement);
    /*if (window.MediaSource) {
        console.log('MediaSource API not supported');
        return;
    }*/
    const mediaSource = new MediaSource();
    // This creates a URL that points to the media buffer,
    // and assigns it to the video element src
    videoElement.src = URL.createObjectURL(mediaSource);
    mediaSource.addEventListener('sourceopen', sourceOpen);
    async function sourceOpen(e: any) {
        URL.revokeObjectURL(videoElement.src);
        console.log('Target: ' + e.target);
        console.log('Ready State:' + mediaSource.readyState);
        const sourceBuffer = mediaSource.addSourceBuffer('video/mp4; codecs="avc1.64002A, mp4a.40.2"');
        sourceBuffer.onupdateend = () => {
            console.log('Ready State:' + mediaSource.readyState);
            // Nothing else to load
            // mediaSource.endOfStream();
            // Start playback!
            // Note: this will fail if video is not muted, due to rules about
            // autoplay and non-muted videos
            videoElement.play();
        };
        axios.get(
            config.Url() +
            '/video/map/PictureBinary/*/Media?mimetypeField=MIMEType&search=Md5=' +
            md5,
            {
                headers: authHeader('video/mp4'),
                responseType: 'arraybuffer',
            }).then((response: any) => {
                const new_blob = new Blob( [ response.data ], { type: 'video/mp4' } );
                console.log('DataX: ' + new_blob);
                videoElement.src = URL.createObjectURL(new_blob);
                // const bytes = new Uint8Array(response.data);
                sourceBuffer.appendBuffer(response.data);
                console.log('Ready State:' + mediaSource.readyState);
                /*const streamData = response.data;
                streamData.on('data', (chunk: ArrayBuffer) => {
                    sourceBuffer.appendBuffer(chunk);
                });
                streamData.on('end', () => {
                    mediaSource.endOfStream();
                    videoElement.play();
                });*/
            },
                (error: any) => console.log('Video stream error ' + md5 + ': ' + error));
    }
}
