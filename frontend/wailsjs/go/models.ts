export namespace model {
	
	export class IesireRow {
	    id: number;
	    productId?: number;
	    pozitie: number;
	    denumire: string;
	    um: string;
	    pretCuTva: number;
	    cantitate: number;
	    pretFaraTva: number;
	    cotaTva: number;
	
	    static createFrom(source: any = {}) {
	        return new IesireRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.productId = source["productId"];
	        this.pozitie = source["pozitie"];
	        this.denumire = source["denumire"];
	        this.um = source["um"];
	        this.pretCuTva = source["pretCuTva"];
	        this.cantitate = source["cantitate"];
	        this.pretFaraTva = source["pretFaraTva"];
	        this.cotaTva = source["cotaTva"];
	    }
	}
	export class IntrareRow {
	    id: number;
	    pozitie: number;
	    denumire: string;
	    um: string;
	    cantitate: number;
	    pretFaraTva: number;
	    pretCuTva: number;
	    cotaTva: number;
	
	    static createFrom(source: any = {}) {
	        return new IntrareRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pozitie = source["pozitie"];
	        this.denumire = source["denumire"];
	        this.um = source["um"];
	        this.cantitate = source["cantitate"];
	        this.pretFaraTva = source["pretFaraTva"];
	        this.pretCuTva = source["pretCuTva"];
	        this.cotaTva = source["cotaTva"];
	    }
	}
	export class Document {
	    id: number;
	    nr: number;
	    data: string;
	    gestiune: string;
	    documentReferinta: string;
	    diferentaTip: string;
	    diferentaValoare: number;
	    incarcaDescarcaTip: string;
	    incarcaDescarcaValoare: number;
	    gestionar: string;
	    calculator: string;
	    vizatCompartimentProductie: string;
	    createdAt: string;
	    updatedAt: string;
	    intrare: IntrareRow[];
	    iesire: IesireRow[];
	
	    static createFrom(source: any = {}) {
	        return new Document(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nr = source["nr"];
	        this.data = source["data"];
	        this.gestiune = source["gestiune"];
	        this.documentReferinta = source["documentReferinta"];
	        this.diferentaTip = source["diferentaTip"];
	        this.diferentaValoare = source["diferentaValoare"];
	        this.incarcaDescarcaTip = source["incarcaDescarcaTip"];
	        this.incarcaDescarcaValoare = source["incarcaDescarcaValoare"];
	        this.gestionar = source["gestionar"];
	        this.calculator = source["calculator"];
	        this.vizatCompartimentProductie = source["vizatCompartimentProductie"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.intrare = this.convertValues(source["intrare"], IntrareRow);
	        this.iesire = this.convertValues(source["iesire"], IesireRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DocumentSummary {
	    id: number;
	    nr: number;
	    data: string;
	    gestiune: string;
	
	    static createFrom(source: any = {}) {
	        return new DocumentSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nr = source["nr"];
	        this.data = source["data"];
	        this.gestiune = source["gestiune"];
	    }
	}
	
	
	export class Product {
	    id: number;
	    denumire: string;
	    um: string;
	    pretCuTva: number;
	    procentDinIntrare: number;
	    ordine: number;
	
	    static createFrom(source: any = {}) {
	        return new Product(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.denumire = source["denumire"];
	        this.um = source["um"];
	        this.pretCuTva = source["pretCuTva"];
	        this.procentDinIntrare = source["procentDinIntrare"];
	        this.ordine = source["ordine"];
	    }
	}
	export class Settings {
	    unitateNume: string;
	    nextNr: number;
	    cotaTva: number;
	    gestiune: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unitateNume = source["unitateNume"];
	        this.nextNr = source["nextNr"];
	        this.cotaTva = source["cotaTva"];
	        this.gestiune = source["gestiune"];
	    }
	}

}

