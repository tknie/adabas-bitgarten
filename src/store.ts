import Vue from 'vue';
import Vuex, { Store } from 'vuex';
import axios from 'axios';
import { config } from './config';
import { authHeader } from './auth-header';
import { userService } from './user.service';
import { image } from './images';
import router from './router';

Vue.use(Vuex);

export default new Vuex.Store({
  state: {
    editorMode: false,
    albums: [{ ISN: 0, Title: '', Thumbnail: '' }],
    albumsData: [{ id: '0' }],
    images: [{ md5: '' }],
    thumbnail: [{ md5: '' }],
    name: 'John Doe',
  },
  getters: {
    NAME: (state) => {
      return state.name;
    },
    Album: (state) => {
      return state.albums;
    },
    getAlbumById: (state, getters) => (id: string) => {
      // console.log('All albums: ' + JSON.stringify(state.albumsData));
      return state.albumsData.find((a) => a.id === id);
    },
    getImageByMd5: (state, getters) => (md5: any) => {
      return state.images.find((a) => a.md5 === md5);
    },
    getThumbnailByMd5: (state, getters) => (md5: any) => {
      // console.log('Get thumbnail ' + state.thumbnail);
      return state.thumbnail.find((a) => a.md5 === md5);
    },
  },
  mutations: {
    CLEAR: (state, name) => {
      state.albums.length = 0;
      state.albumsData.length = 0;
      state.images.length = 0;
      state.thumbnail.length = 0;
    },
    SET_NAME: (state, name) => {
      state.name = name;
    },
    SET_ALBUMS: (state, albums) => {
      state.albums.length = 0;
      albums.forEach((a: any) => {
        const d = new Date(a.Date * 1000);
        const x = {
          ISN: a.ISN, Title: a.Title, Date: d, DateTime: a.Date,
          Thumbnail: a.Metadata.Thumbnail,
        };
        state.albums.push(x);
        const sorted = state.albums.sort((a1: any, b: any) => {
          if (a1.DateTime < b.DateTime) {
            return -1;
          }
          if (a1.DateTime > b.DateTime) {
            return 1;
          }
          return 0;
        });
        //        console.log("X" + JSON.stringify(sorted));
        state.albums = sorted;
      });
      //      state.albums = albums;
    },
    ADD_ALBUM: (state, album) => {
      state.albumsData.push(album);
    },
    ADD_IMAGE: (state, img) => {
      state.images.push(img);
    },
    ADD_THUMB: (state, thumb) => {
      state.thumbnail.push(thumb);
    },
  },
  actions: {
    INIT_ALBUMS: async (context, name) => {
      const getConfig = {
        headers: authHeader('application/json'),
        useCredentails: true,
      };
      // console.log('Init receiving Albums');
      await axios.get(config.Url() + '/rest/map/Albums?fields=Title,Date,Thumbnail&limit=0',
        getConfig).then((response: any) => {
          // console.log('Receiving Albums ' + response.status);
          if (response.status !== 200) {
            console.log('Error receiving Albums ' + response.status);
            if (response.status === 401 || response.status === 404) {
              // auto logout if 401 response returned from api
              userService.logout();
              location.reload();
            }

            const error = response.statusText;
            return Promise.reject(error);
          }
          if (response === undefined) {
            console.log('Response undefined ...' + response.text());
            return;
          }
          if (response.data === undefined) {
            console.log('Response data undefined ...' + response.text());
            console.log('Error receiving Albums: ' + JSON.stringify(response));
            return;
          }
          // console.log('Got receiving Albums: ' + JSON.stringify(response));
          response.data.Records.forEach((r: any) => {
            context.dispatch('LOAD_THUMB', r.Metadata.Thumbnail);
          });
          context.commit('SET_ALBUMS', response.data.Records);
        })
        .catch((error: any) => {
          console.log('Error receiving Albums' + JSON.stringify(error));
          userService.logout();
          if (router.currentRoute.name !== 'login') {
            location.reload();
          }
        });
    },
    INIT_ALBUM: async (context, x) => {
      if (!x.nr) {
        console.log('Nr wrong' + x.nr);
        return;
      }
      const getConfig = {
        headers: authHeader('application/json'),
        useCredentails: true,
      };
      // console.log('Init album ' + x.nr);
      await axios.get(`${config.Url()}/rest/map/Album/${x.nr}`,
        getConfig).then((response) => {
          if (response.status !== 200) {
            console.log('Error loading album ' + x.nr + ':' + response.status);
            if (response.status === 401 || response.status === 404) {
              // auto logout if 401 response returned from api
              userService.logout();
              location.reload();
            }

            const error = response.statusText;
            return Promise.reject(error);
          }
          // console.log('Loading album ' + x.nr);
          const p: any[] = [];
          const record = response.data.Records[0];
          record.Pictures.forEach((element: any, index: number) => {
            // console.log('Element: '+JSON.stringify(element));
            const i = {
              index: index + 1,
              src: element.Md5,
              msrc: element.Md5,
              MIMEType: element.MIMEType,
              fill: element.Fill,
              w: element.Size.Width,
              h: element.Size.Height,
              title: element.Description,
            };
            if (x.loadImage) {
              if (i.MIMEType.startsWith('image/')) {
                // console.log('Load image '+element.Name+' '+i.MIMEType)
                context.dispatch('LOAD_IMAGE', i.src);
              } else {
                // console.log('Load video '+element.Name+' '+i.MIMEType)
                // context.dispatch('LOAD_VIDEO', i.src);
              }
            }
            // context.dispatch('LOAD_THUMB', i.msrc);
            p.push(i);
          });
          const album = {
            id: x.nr, Title: record.Title,
            date: record.Date, pictures: p,
          };
          context.commit('ADD_ALBUM', album);
        },
          (error) => {
            userService.logout();
            location.reload();
          });
    },
    LOAD_THUMBS: async (context, x) => {
      if (!x.nr) {
        console.log('Nr wrong' + x.nr);
        return;
      }

      const getConfig = {
        headers: authHeader('application/json'),
        useCredentails: true,
      };
      console.log('Init thumbnails ' + x.nr);
      await axios.get(`${config.Url()}/rest/map/Album/${x.nr}`,
        getConfig).then((response) => {
          if (response.status !== 200) {
            console.log('Error loading album ' + x.nr + ':' + response.status);
            if (response.status === 401 || response.status === 404) {
              // auto logout if 401 response returned from api
              userService.logout();
              location.reload();
            }

            const error = response.statusText;
            return Promise.reject(error);
          }
          // console.log('Loading album ' + x.nr);
          const p: any[] = [];
          const record = response.data.Records[0];
          record.Pictures.forEach((element: any, index: number) => {
            // console.log('Element: '+JSON.stringify(element));
            const i = {
              index: index + 1,
              src: element.Md5,
              msrc: element.Md5,
              MIMEType: element.MIMEType,
              fill: element.Fill,
              w: element.Size.Width,
              h: element.Size.Height,
              title: element.Description,
            };
            context.dispatch('LOAD_THUMB', i.msrc);
            p.push(i);
          });
          const album = {
            id: x.nr, Title: record.Title,
            date: record.Date, pictures: p,
          };
          context.commit('ADD_ALBUM', album);
        },
          (error) => {
            userService.logout();
            location.reload();
          });
    },
    LOAD_IMAGE: async (context, md5) => {
      if (!md5) {
        return;
      }
      const x = context.getters.getImageByMd5(md5);
      if (x) {
        return;
      }
      // console.log('Init load image ' + md5);
      const getConfig = {
        headers: authHeader('application/json'),
      };
      await image.loadImage(md5).then((response) => {
        if ((response) && (response.data)) {
          const img = new Image();
          const i = { md5, width: 0, height: 0, fill: 'fill', src: response.data, time: new Date() };
          img.onload = () => {
            i.width = img.width;
            i.height = img.height;
            // console.log('Onload '+img.width+' '+img.height+' '+i.md5);
            // console.log(md5 + ' image loaded');
            context.commit('ADD_IMAGE', i);
          };
        }
      },
        (error) => console.log('Error loading image' + error));
    },
    LOAD_VIDEO: async (context, md5) => {
      if (!md5) {
        return;
      }
      const x = context.getters.getImageByMd5(md5);
      if (x) {
        return;
      }
      // console.log('Init load image ' + md5);
      const getConfig = {
        headers: authHeader('application/json'),
      };
      await image.loadVideo(md5).then((response) => {
        if ((response) && (response.data)) {
          const i = {
            md5, width: 0, height: 0, fill: 'fillHeight',
            MIMEType: 'video/mp4', src: response.data, time: new Date(),
          };
          context.commit('ADD_IMAGE', i);
        }
      },
        (error) => console.log('Error loading video' + error));
    },
    LOAD_THUMB: async (context, md5) => {
      const x = context.getters.getThumbnailByMd5(md5);
      if (x) {
        return x;
      }
      // console.log('Store load thumbnail ' + md5);
      const getConfig = {
        headers: authHeader('application/json'),
      };
      await image.loadThumbnail(md5).then((response) => {
        if ((response) && (response.data)) {
          const th = { md5, src: response.data };
          context.commit('ADD_THUMB', th);
          return response.data;
        }
        const thumb = { md5, src: response };
        context.commit('ADD_THUMB', thumb);
        return response;
      });
    },
  },
});
