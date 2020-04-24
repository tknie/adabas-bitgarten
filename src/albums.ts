import store from './store';
import { authHeader } from './auth-header';
import { config } from './config';

export const albums = {
    loadAlbums,
    storeAlbums,
    updateAlbums,
    deleteAlbum,
};

export function loadAlbums() {
    let a = store.getters.Albums;
    if (!a) {
        store.dispatch('INIT_ALBUM', {nr: 0});
        a = store.getters.Albums;
    }
    return a;
}

export function storeAlbums(album: any) {
    const h = authHeader('application/json');
    const s = {Store: [album]};
    console.log('Store ' + JSON.stringify(s));
    const requestOptions = {
            method: 'POST',
            headers: { ContentType: 'application/json',
               Authorization: h.Authorization,
               Accept: h.Accept,
            },
            body: JSON.stringify(s),
    };

    return fetch(`${config.Url()}/rest/map/Album`, requestOptions)
    .then((res: any) => console.log(res));
}

export function updateAlbums(isn: number, album: any) {
    const h = authHeader('application/json');
    const s = {Store: [album]};
    console.log('Update ' + JSON.stringify(s));
    const requestOptions = {
            method: 'PUT',
            headers: { ContentType: 'application/json',
               Authorization: h.Authorization,
               Accept: h.Accept,
            },
            body: JSON.stringify(s),
    };

    return fetch(`${config.Url()}/rest/map/Album/` + isn + `?exchange=true`, requestOptions)
    .then((res: any) => console.log(res));
}

export function deleteAlbum(isn: number) {
    if (isn === 0) {
        return;
    }
    const h = authHeader('application/json');
    console.log('Delete ' + isn);
    const requestOptions = {
            method: 'DELETE',
            headers: authHeader('application/json'),
    };

    return fetch(`${config.Url()}/rest/map/Album/` + isn, requestOptions)
    .then((res: any) => console.log(res));
}
